use std::borrow::Cow;
use wgpu::util::DeviceExt;
use serde::{Serialize, Deserialize};
use std::path::Path;
use tokio::io::AsyncBufReadExt;

const MAX_VMS: usize = 4096;
const MAX_SPAWNS_PER_PASS: usize = 1024;
const MAX_PASSES: usize = 8;
const MAX_STACK: usize = 256;
const MAX_VARS: usize = 64;

#[repr(C)]
#[derive(Copy, Clone, Debug, bytemuck::Pod, bytemuck::Zeroable, Serialize, Deserialize)]
struct VMState {
    pc: u32,
    sp: u32,
    halted: u32,
    error: u32,
    steps: u32,
    result_tag: u32,
    result_data: i32,
    pad: u32,
}

#[repr(C)]
#[derive(Copy, Clone, Debug, bytemuck::Pod, bytemuck::Zeroable, Serialize, Deserialize)]
struct GpuValue {
    tag: u32,
    data: i32,
}

#[repr(C)]
#[derive(Copy, Clone, Debug, bytemuck::Pod, bytemuck::Zeroable)]
struct SpawnRequest {
    parent_id: u32,
    pc_offset: i32,
}

#[derive(Deserialize)]
struct GlyphJob {
    bytecode_hex: String,
    num_vms: u32,
    workgroup_count: u32,
    code_offset: u32,
    const_offset: u32,
    num_constants: u32,
}

#[repr(C)]
#[derive(Copy, Clone, Debug, bytemuck::Pod, bytemuck::Zeroable)]
struct Config {
    bytecode_len: u32,
    num_constants: u32,
    constants_offset: u32,
    code_offset: u32,
    num_vms: u32,
    pad1: u32,
    pad2: u32,
    pad3: u32,
}

#[tokio::main]
async fn main() {
    let mut args = std::env::args().skip(1);
    let shader_path_str = args.next().expect("No WGSL shader path provided");
    let shader_path = Path::new(&shader_path_str);
    
    let is_daemon = std::env::var("GLYPH_DAEMON").is_ok();

    let instance = wgpu::Instance::default();
    let adapter = instance.request_adapter(&wgpu::RequestAdapterOptions {
        power_preference: wgpu::PowerPreference::HighPerformance,
        ..Default::default()
    }).await.expect("No suitable GPU adapter found");
    
    let (device, queue) = adapter.request_device(&wgpu::DeviceDescriptor {
        label: None,
        required_features: wgpu::Features::empty(),
        required_limits: wgpu::Limits::default(),
    }, None).await.expect("Failed to create device");

    if is_daemon {
        eprintln!("[GPU] Starting IPC daemon mode...");
        let mut lines = tokio::io::BufReader::new(tokio::io::stdin()).lines();
        
        while let Ok(Some(line)) = lines.next_line().await {
            let job: GlyphJob = match serde_json::from_str(&line) {
                Ok(j) => j,
                Err(e) => {
                    eprintln!("[GPU] Invalid job JSON: {}", e);
                    continue;
                }
            };

            let bytecode = match hex::decode(&job.bytecode_hex) {
                Ok(b) => b,
                Err(e) => {
                    eprintln!("[GPU] Invalid hex bytecode: {}", e);
                    continue;
                }
            };

            let wgsl_source = match std::fs::read_to_string(shader_path) {
                Ok(s) => s,
                Err(e) => {
                    eprintln!("[GPU] Failed to read shader: {}", e);
                    continue;
                }
            };
            
            let shader = device.create_shader_module(wgpu::ShaderModuleDescriptor {
                label: Some("GlyphLang Shader"),
                source: wgpu::ShaderSource::Wgsl(Cow::Borrowed(&wgsl_source)),
            });

            let compute_pipeline = device.create_compute_pipeline(&wgpu::ComputePipelineDescriptor {
                label: Some("Compute Pipeline"),
                layout: None,
                module: &shader,
                entry_point: "main",
            });

            let mut current_vms = job.num_vms as usize;
            
            let config = Config {
                bytecode_len: bytecode.len() as u32,
                num_constants: job.num_constants,
                constants_offset: job.const_offset,
                code_offset: job.code_offset,
                num_vms: MAX_VMS as u32,
                ..bytemuck::Zeroable::zeroed()
            };
            let config_buffer = device.create_buffer_init(&wgpu::util::BufferInitDescriptor {
                label: Some("Config Buffer"),
                contents: bytemuck::cast_slice(&[config]),
                usage: wgpu::BufferUsages::STORAGE | wgpu::BufferUsages::COPY_DST,
            });

            let mut bytecode_u32 = Vec::new();
            for chunk in bytecode.chunks(4) {
                let mut word = [0u8; 4];
                word[..chunk.len()].copy_from_slice(chunk);
                bytecode_u32.push(u32::from_le_bytes(word));
            }
            let bytecode_buffer = device.create_buffer_init(&wgpu::util::BufferInitDescriptor {
                label: Some("Bytecode Buffer"),
                contents: bytemuck::cast_slice(&bytecode_u32),
                usage: wgpu::BufferUsages::STORAGE | wgpu::BufferUsages::COPY_SRC | wgpu::BufferUsages::COPY_DST,
            });

            let initial_states = vec![VMState { pc: 0, sp: 0, halted: 0, error: 0, steps: 0, result_tag: 0, result_data: 0, pad: 0 }; MAX_VMS];
            let state_buffer = device.create_buffer(&wgpu::BufferDescriptor {
                label: Some("VMState Buffer"),
                size: (std::mem::size_of::<VMState>() * MAX_VMS) as u64,
                usage: wgpu::BufferUsages::STORAGE | wgpu::BufferUsages::COPY_SRC | wgpu::BufferUsages::COPY_DST,
                mapped_at_creation: false,
            });
            queue.write_buffer(&state_buffer, 0, bytemuck::cast_slice(&initial_states[..current_vms]));

            let stacks_buffer = device.create_buffer(&wgpu::BufferDescriptor {
                label: Some("Stacks Buffer"),
                size: (std::mem::size_of::<GpuValue>() * MAX_STACK * MAX_VMS) as u64,
                usage: wgpu::BufferUsages::STORAGE | wgpu::BufferUsages::COPY_SRC | wgpu::BufferUsages::COPY_DST,
                mapped_at_creation: false,
            });
            let zero_stack = vec![0u8; std::mem::size_of::<GpuValue>() * MAX_STACK * MAX_VMS];
            queue.write_buffer(&stacks_buffer, 0, &zero_stack);

            let vars_buffer = device.create_buffer(&wgpu::BufferDescriptor {
                label: Some("Vars Buffer"),
                size: (std::mem::size_of::<GpuValue>() * MAX_VARS * MAX_VMS) as u64,
                usage: wgpu::BufferUsages::STORAGE | wgpu::BufferUsages::COPY_SRC | wgpu::BufferUsages::COPY_DST,
                mapped_at_creation: false,
            });
            let zero_vars = vec![0u8; std::mem::size_of::<GpuValue>() * MAX_VARS * MAX_VMS];
            queue.write_buffer(&vars_buffer, 0, &zero_vars);

            let spawn_buffer_size = 4 + (std::mem::size_of::<SpawnRequest>() * MAX_SPAWNS_PER_PASS);
            let spawn_buffer = device.create_buffer(&wgpu::BufferDescriptor {
                label: Some("Spawn Buffer"),
                size: spawn_buffer_size as u64,
                usage: wgpu::BufferUsages::STORAGE | wgpu::BufferUsages::COPY_SRC | wgpu::BufferUsages::COPY_DST,
                mapped_at_creation: false,
            });

            let readback_spawn_buffer = device.create_buffer(&wgpu::BufferDescriptor {
                label: Some("Readback Spawn Buffer"),
                size: spawn_buffer_size as u64,
                usage: wgpu::BufferUsages::COPY_DST | wgpu::BufferUsages::MAP_READ,
                mapped_at_creation: false,
            });

            let readback_state_buffer = device.create_buffer(&wgpu::BufferDescriptor {
                label: Some("Readback State Buffer"),
                size: (std::mem::size_of::<VMState>() * MAX_VMS) as u64,
                usage: wgpu::BufferUsages::COPY_DST | wgpu::BufferUsages::MAP_READ,
                mapped_at_creation: false,
            });

            let readback_stacks_buffer = device.create_buffer(&wgpu::BufferDescriptor {
                label: Some("Readback Stacks Buffer"),
                size: (std::mem::size_of::<GpuValue>() * MAX_STACK * MAX_VMS) as u64,
                usage: wgpu::BufferUsages::COPY_DST | wgpu::BufferUsages::MAP_READ,
                mapped_at_creation: false,
            });

            let readback_vars_buffer = device.create_buffer(&wgpu::BufferDescriptor {
                label: Some("Readback Vars Buffer"),
                size: (std::mem::size_of::<GpuValue>() * MAX_VARS * MAX_VMS) as u64,
                usage: wgpu::BufferUsages::COPY_DST | wgpu::BufferUsages::MAP_READ,
                mapped_at_creation: false,
            });

            let bind_group_layout = compute_pipeline.get_bind_group_layout(0);
            let bind_group = device.create_bind_group(&wgpu::BindGroupDescriptor {
                label: None,
                layout: &bind_group_layout,
                entries: &[
                    wgpu::BindGroupEntry { binding: 0, resource: config_buffer.as_entire_binding() },
                    wgpu::BindGroupEntry { binding: 1, resource: bytecode_buffer.as_entire_binding() },
                    wgpu::BindGroupEntry { binding: 2, resource: state_buffer.as_entire_binding() },
                    wgpu::BindGroupEntry { binding: 3, resource: stacks_buffer.as_entire_binding() },
                    wgpu::BindGroupEntry { binding: 4, resource: vars_buffer.as_entire_binding() },
                    wgpu::BindGroupEntry { binding: 5, resource: spawn_buffer.as_entire_binding() },
                ],
            });

            for _pass in 0..MAX_PASSES {
                queue.write_buffer(&spawn_buffer, 0, bytemuck::cast_slice(&[0u32]));

                let mut encoder = device.create_command_encoder(&wgpu::CommandEncoderDescriptor { label: None });
                {
                    let mut cpass = encoder.begin_compute_pass(&wgpu::ComputePassDescriptor { label: None, timestamp_writes: None });
                    cpass.set_pipeline(&compute_pipeline);
                    cpass.set_bind_group(0, &bind_group, &[]);
                    let wgs = (current_vms + 63) / 64;
                    cpass.dispatch_workgroups(wgs as u32, 1, 1);
                }
                queue.submit(Some(encoder.finish()));

                let mut encoder = device.create_command_encoder(&wgpu::CommandEncoderDescriptor { label: None });
                encoder.copy_buffer_to_buffer(&spawn_buffer, 0, &readback_spawn_buffer, 0, spawn_buffer_size as u64);
                queue.submit(Some(encoder.finish()));

                let (s, r) = futures_intrusive::channel::shared::oneshot_channel();
                readback_spawn_buffer.slice(..).map_async(wgpu::MapMode::Read, move |v| s.send(v).unwrap());
                device.poll(wgpu::Maintain::Wait);

                let spawn_count = if let Some(Ok(())) = r.receive().await {
                    let data = readback_spawn_buffer.slice(..).get_mapped_range();
                    let count = *bytemuck::from_bytes::<u32>(&data[0..4]);
                    drop(data);
                    readback_spawn_buffer.unmap();
                    count as usize
                } else { 0 };

                if spawn_count == 0 || current_vms >= MAX_VMS {
                    break;
                }

                let mut encoder = device.create_command_encoder(&wgpu::CommandEncoderDescriptor { label: None });
                encoder.copy_buffer_to_buffer(&state_buffer, 0, &readback_state_buffer, 0, (std::mem::size_of::<VMState>() * current_vms) as u64);
                encoder.copy_buffer_to_buffer(&stacks_buffer, 0, &readback_stacks_buffer, 0, (std::mem::size_of::<GpuValue>() * MAX_STACK * current_vms) as u64);
                encoder.copy_buffer_to_buffer(&vars_buffer, 0, &readback_vars_buffer, 0, (std::mem::size_of::<GpuValue>() * MAX_VARS * current_vms) as u64);
                queue.submit(Some(encoder.finish()));

                let (s1, r1) = futures_intrusive::channel::shared::oneshot_channel();
                let (s2, r2) = futures_intrusive::channel::shared::oneshot_channel();
                let (s3, r3) = futures_intrusive::channel::shared::oneshot_channel();
                let (s4, r4) = futures_intrusive::channel::shared::oneshot_channel();
                
                readback_state_buffer.slice(..).map_async(wgpu::MapMode::Read, move |v| s1.send(v).unwrap());
                readback_stacks_buffer.slice(..).map_async(wgpu::MapMode::Read, move |v| s2.send(v).unwrap());
                readback_vars_buffer.slice(..).map_async(wgpu::MapMode::Read, move |v| s3.send(v).unwrap());
                readback_spawn_buffer.slice(..).map_async(wgpu::MapMode::Read, move |v| s4.send(v).unwrap());
                device.poll(wgpu::Maintain::Wait);

                if let (Some(Ok(())), Some(Ok(())), Some(Ok(())), Some(Ok(()))) = (r1.receive().await, r2.receive().await, r3.receive().await, r4.receive().await) {
                    let state_data = readback_state_buffer.slice(..).get_mapped_range();
                    let current_states: &[VMState] = bytemuck::cast_slice(&state_data);
                    
                    let stacks_data = readback_stacks_buffer.slice(..).get_mapped_range();
                    let current_stacks: &[GpuValue] = bytemuck::cast_slice(&stacks_data);
                    
                    let vars_data = readback_vars_buffer.slice(..).get_mapped_range();
                    let current_vars: &[GpuValue] = bytemuck::cast_slice(&vars_data);
                    
                    let spawn_data = readback_spawn_buffer.slice(..).get_mapped_range();
                    let requests: &[SpawnRequest] = bytemuck::cast_slice(&spawn_data[4..]);
                    
                    let mut new_states = Vec::new();
                    for i in 0..spawn_count.min(MAX_SPAWNS_PER_PASS).min(MAX_VMS - current_vms) {
                        let req = requests[i];
                        if (req.parent_id as usize) < current_vms {
                            let parent = current_states[req.parent_id as usize];
                            new_states.push(VMState {
                                pc: (parent.pc as i32 + req.pc_offset) as u32,
                                sp: parent.sp,
                                halted: 0,
                                error: 0,
                                steps: 0,
                                result_tag: 0,
                                result_data: 0,
                                pad: 0,
                            });
                            
                            let parent_stack = &current_stacks[req.parent_id as usize * MAX_STACK..(req.parent_id as usize + 1) * MAX_STACK];
                            let parent_vars = &current_vars[req.parent_id as usize * MAX_VARS..(req.parent_id as usize + 1) * MAX_VARS];
                            
                            queue.write_buffer(&stacks_buffer, ((current_vms + i) * MAX_STACK * std::mem::size_of::<GpuValue>()) as u64, bytemuck::cast_slice(parent_stack));
                            queue.write_buffer(&vars_buffer, ((current_vms + i) * MAX_VARS * std::mem::size_of::<GpuValue>()) as u64, bytemuck::cast_slice(parent_vars));
                        }
                    }
                    drop(state_data); drop(stacks_data); drop(vars_data); drop(spawn_data);
                    readback_state_buffer.unmap(); readback_stacks_buffer.unmap(); readback_vars_buffer.unmap(); readback_spawn_buffer.unmap();

                    if !new_states.is_empty() {
                        queue.write_buffer(&state_buffer, (std::mem::size_of::<VMState>() * current_vms) as u64, bytemuck::cast_slice(&new_states));
                        current_vms += new_states.len();
                    }
                } else {
                    readback_state_buffer.unmap(); readback_stacks_buffer.unmap(); readback_vars_buffer.unmap(); readback_spawn_buffer.unmap();
                    break;
                }
            }

            let mut encoder = device.create_command_encoder(&wgpu::CommandEncoderDescriptor { label: None });
            encoder.copy_buffer_to_buffer(&state_buffer, 0, &readback_state_buffer, 0, (std::mem::size_of::<VMState>() * current_vms) as u64);
            queue.submit(Some(encoder.finish()));

            let (s, r) = futures_intrusive::channel::shared::oneshot_channel();
            readback_state_buffer.slice(..).map_async(wgpu::MapMode::Read, move |v| s.send(v).unwrap());
            device.poll(wgpu::Maintain::Wait);

            if let Some(Ok(())) = r.receive().await {
                let data = readback_state_buffer.slice(..).get_mapped_range();
                let results: &[VMState] = bytemuck::cast_slice(&data);
                println!("{}", serde_json::to_string(&results[..current_vms]).unwrap());
                drop(data);
                readback_state_buffer.unmap();
            }
        }
    }
}
