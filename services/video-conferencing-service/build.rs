fn main() -> Result<(), Box<dyn std::error::Error>> {
    let out_dir = std::env::var("OUT_DIR").unwrap();
    tonic_build::configure()
        .build_server(true)
        .build_client(true)
        .file_descriptor_set_path(format!("{}/video_conferencing_descriptor.bin", out_dir))
        .compile_protos(
            &["../../proto/video_conferencing.proto"],
            &["../../proto"],
        )?;
    Ok(())
}
