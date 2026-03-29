use std::borrow::Cow;
use wgpu::util::DeviceExt;
use serde::{Serialize, Deserialize};
use notify::{Watcher, RecursiveMode, Event};
use std::path::Path;
use tokio::sync::mpsc;

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

async fn run() {
    let mut args = std::env::args().skip(1);
    let shader_path_str = args.next().expect("No WGSL shader path provided");
    let shader_path = Path::new(&shader_path_str);
    
    let num_vms: u32 = args.next().map(|s| s.parse().unwrap()).unwrap_or(1);
    let workgroup_count: u32 = args.next().map(|s| s.parse().unwrap()).unwrap_or(1);
    let is_watch = std::env::var("GLYPH_WATCH").is_ok();

    // 1. Setup WGPU
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

    // 2. Persistent Buffers
    let initial_states = vec![VMState {
        pc: 0, sp: 0, halted: 0, error: 0, steps: 0, result_tag: 0, result_data: 0, pad: 0,
    }; num_vms as usize];
    
    let state_buffer = device.create_buffer_init(&wgpu::util::BufferInitDescriptor {
        label: Some("VMState Buffer"),
        contents: bytemuck::cast_slice(&initial_states),
        usage: wgpu::BufferUsages::STORAGE | wgpu::BufferUsages::COPY_SRC | wgpu::BufferUsages::COPY_DST,
    });

    let locals = vec![0i32; 64 * num_vms as usize];
    let locals_buffer = device.create_buffer_init(&wgpu::util::BufferInitDescriptor {
        label: Some("Locals Buffer"),
        contents: bytemuck::cast_slice(&locals),
        usage: wgpu::BufferUsages::STORAGE | wgpu::BufferUsages::COPY_SRC | wgpu::BufferUsages::COPY_DST,
    });

    // 3. Hot Reload Loop
    let (tx, mut rx) = mpsc::channel(1);
    let mut watcher = notify::recommended_watcher(move |res: notify::Result<Event>| {
        if let Ok(event) = res {
            if event.kind.is_modify() {
                let _ = tx.blocking_send(());
            }
        }
    }).unwrap();

    if is_watch {
        watcher.watch(shader_path, RecursiveMode::NonRecursive).unwrap();
        eprintln!("[GPU] Watching {} for hot-reload...", shader_path_str);
    }

    loop {
        let wgsl_source = std::fs::read_to_string(shader_path).expect("Failed to read shader file");
        
        // 4. Create Pipeline
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

        let bind_group_layout = compute_pipeline.get_bind_group_layout(0);
        let bind_group = device.create_bind_group(&wgpu::BindGroupDescriptor {
            label: None,
            layout: &bind_group_layout,
            entries: &[
                wgpu::BindGroupEntry {
                    binding: 0,
                    resource: state_buffer.as_entire_binding(),
                },
                wgpu::BindGroupEntry {
                    binding: 1,
                    resource: locals_buffer.as_entire_binding(),
                },
            ],
        });

        // 5. Dispatch
        let mut encoder = device.create_command_encoder(&wgpu::CommandEncoderDescriptor { label: None });
        {
            let mut cpass = encoder.begin_compute_pass(&wgpu::ComputePassDescriptor { label: None, timestamp_writes: None });
            cpass.set_pipeline(&compute_pipeline);
            cpass.set_bind_group(0, &bind_group, &[]);
            cpass.dispatch_workgroups(workgroup_count, 1, 1);
        }
        queue.submit(Some(encoder.finish()));

        // 6. Report Result (simplified for hot-reload)
        if !is_watch {
            // Read back and exit if not in watch mode
            let readback_buffer = device.create_buffer(&wgpu::BufferDescriptor {
                label: Some("Readback Buffer"),
                size: (std::mem::size_of::<VMState>() * num_vms as usize) as u64,
                usage: wgpu::BufferUsages::COPY_DST | wgpu::BufferUsages::MAP_READ,
                mapped_at_creation: false,
            });

            let mut encoder = device.create_command_encoder(&wgpu::CommandEncoderDescriptor { label: None });
            encoder.copy_buffer_to_buffer(&state_buffer, 0, &readback_buffer, 0, (std::mem::size_of::<VMState>() * num_vms as usize) as u64);
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
            break;
        }

        eprintln!("[GPU] Dispatch complete. result_data is now in vm_stats. Waiting for changes...");
        // Wait for file change
        rx.recv().await;
        eprintln!("[GPU] Reloading shader...");
    }
}

#[tokio::main]
async fn main() {
    run().await;
}
