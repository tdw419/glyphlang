use std::borrow::Cow;
use wgpu::util::DeviceExt;
use serde::{Serialize, Deserialize};

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
    let wgsl_source = std::env::args().nth(1).expect("No WGSL source provided");
    
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

    // 2. Create Shader
    let shader = device.create_shader_module(wgpu::ShaderModuleDescriptor {
        label: Some("GlyphLang Shader"),
        source: wgpu::ShaderSource::Wgsl(Cow::Borrowed(&wgsl_source)),
    });

    // 3. Create Buffers
    let initial_state = VMState {
        pc: 0, sp: 0, halted: 0, error: 0, steps: 0, result_tag: 0, result_data: 0, pad: 0,
    };
    
    let state_buffer = device.create_buffer_init(&wgpu::util::BufferInitDescriptor {
        label: Some("VMState Buffer"),
        contents: bytemuck::cast_slice(&[initial_state]),
        usage: wgpu::BufferUsages::STORAGE | wgpu::BufferUsages::COPY_SRC | wgpu::BufferUsages::COPY_DST,
    });

    let locals = vec![0i32; 64]; // Match 64 locals per VM
    let locals_buffer = device.create_buffer_init(&wgpu::util::BufferInitDescriptor {
        label: Some("Locals Buffer"),
        contents: bytemuck::cast_slice(&locals),
        usage: wgpu::BufferUsages::STORAGE | wgpu::BufferUsages::COPY_SRC | wgpu::BufferUsages::COPY_DST,
    });

    // 4. Create Pipeline
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
        cpass.dispatch_workgroups(1, 1, 1);
    }
    
    queue.submit(Some(encoder.finish()));

    // 6. Read back results
    let readback_buffer = device.create_buffer(&wgpu::BufferDescriptor {
        label: Some("Readback Buffer"),
        size: std::mem::size_of::<VMState>() as u64,
        usage: wgpu::BufferUsages::COPY_DST | wgpu::BufferUsages::MAP_READ,
        mapped_at_creation: false,
    });

    let mut encoder = device.create_command_encoder(&wgpu::CommandEncoderDescriptor { label: None });
    encoder.copy_buffer_to_buffer(&state_buffer, 0, &readback_buffer, 0, std::mem::size_of::<VMState>() as u64);
    queue.submit(Some(encoder.finish()));

    let buffer_slice = readback_buffer.slice(..);
    let (sender, receiver) = futures_intrusive::channel::shared::oneshot_channel();
    buffer_slice.map_async(wgpu::MapMode::Read, move |v| sender.send(v).unwrap());

    device.poll(wgpu::Maintain::Wait);

    if let Some(Ok(())) = receiver.receive().await {
        let data = buffer_slice.get_mapped_range();
        let result: VMState = *bytemuck::from_bytes(&data);
        println!("{}", serde_json::to_string(&result).unwrap());
    } else {
        panic!("Failed to read buffer from GPU");
    }
}

fn main() {
    pollster::block_on(run());
}
