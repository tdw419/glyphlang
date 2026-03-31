use std::borrow::Cow;
use wgpu::util::DeviceExt;
use serde::{Serialize, Deserialize};
use std::path::Path;
use tokio::io::AsyncBufReadExt;

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
}

async fn run() {
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

            let _bytecode = match hex::decode(&job.bytecode_hex) {
                Ok(b) => b,
                Err(e) => {
                    eprintln!("[GPU] Invalid hex bytecode: {}", e);
                    continue;
                }
            };

            let wgsl_source = std::fs::read_to_string(shader_path).expect("Failed to read shader file");
            
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

            let initial_states = vec![VMState {
                pc: 0, sp: 0, halted: 0, error: 0, steps: 0, result_tag: 0, result_data: 0, pad: 0,
            }; job.num_vms as usize];
            
            let state_buffer = device.create_buffer_init(&wgpu::util::BufferInitDescriptor {
                label: Some("VMState Buffer"),
                contents: bytemuck::cast_slice(&initial_states),
                usage: wgpu::BufferUsages::STORAGE | wgpu::BufferUsages::COPY_SRC | wgpu::BufferUsages::COPY_DST,
            });

            let locals = vec![0i32; 64 * job.num_vms as usize];
            let locals_buffer = device.create_buffer_init(&wgpu::util::BufferInitDescriptor {
                label: Some("Locals Buffer"),
                contents: bytemuck::cast_slice(&locals),
                usage: wgpu::BufferUsages::STORAGE | wgpu::BufferUsages::COPY_SRC | wgpu::BufferUsages::COPY_DST,
            });

            let spawn_buffer_size = 4 + (8 * 1024);
            let spawn_buffer = device.create_buffer(&wgpu::BufferDescriptor {
                label: Some("Spawn Buffer"),
                size: spawn_buffer_size,
                usage: wgpu::BufferUsages::STORAGE | wgpu::BufferUsages::COPY_SRC | wgpu::BufferUsages::COPY_DST,
                mapped_at_creation: false,
            });
            queue.write_buffer(&spawn_buffer, 0, bytemuck::cast_slice(&[0u32]));

            let bind_group_layout = compute_pipeline.get_bind_group_layout(0);
            let bind_group = device.create_bind_group(&wgpu::BindGroupDescriptor {
                label: None,
                layout: &bind_group_layout,
                entries: &[
                    wgpu::BindGroupEntry { binding: 0, resource: state_buffer.as_entire_binding() },
                    wgpu::BindGroupEntry { binding: 1, resource: locals_buffer.as_entire_binding() },
                    wgpu::BindGroupEntry { binding: 5, resource: spawn_buffer.as_entire_binding() },
                ],
            });

            let mut encoder = device.create_command_encoder(&wgpu::CommandEncoderDescriptor { label: None });
            {
                let mut cpass = encoder.begin_compute_pass(&wgpu::ComputePassDescriptor { label: None, timestamp_writes: None });
                cpass.set_pipeline(&compute_pipeline);
                cpass.set_bind_group(0, &bind_group, &[]);
                cpass.dispatch_workgroups(job.workgroup_count, 1, 1);
            }
            queue.submit(Some(encoder.finish()));

            let readback_buffer = device.create_buffer(&wgpu::BufferDescriptor {
                label: Some("Readback Buffer"),
                size: (std::mem::size_of::<VMState>() * job.num_vms as usize) as u64,
                usage: wgpu::BufferUsages::COPY_DST | wgpu::BufferUsages::MAP_READ,
                mapped_at_creation: false,
            });

            let mut encoder = device.create_command_encoder(&wgpu::CommandEncoderDescriptor { label: None });
            encoder.copy_buffer_to_buffer(&state_buffer, 0, &readback_buffer, 0, (std::mem::size_of::<VMState>() * job.num_vms as usize) as u64);
            queue.submit(Some(encoder.finish()));

            let buffer_slice = readback_buffer.slice(..);
            let (s, r) = futures_intrusive::channel::shared::oneshot_channel();
            buffer_slice.map_async(wgpu::MapMode::Read, move |v| s.send(v).unwrap());
            device.poll(wgpu::Maintain::Wait);

            if let Some(Ok(())) = r.receive().await {
                let data = buffer_slice.get_mapped_range();
                let results: &[VMState] = bytemuck::cast_slice(&data);
                println!("{}", serde_json::to_string(&results).unwrap());
            }
        }
    }
}

fn main() {
    pollster::block_on(run());
}
