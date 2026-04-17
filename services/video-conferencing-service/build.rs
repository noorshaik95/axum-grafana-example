fn main() -> Result<(), Box<dyn std::error::Error>> {
    tonic_build::configure()
        .build_server(true)
        .build_client(true)
        .file_descriptor_set_path("target/video_conferencing_descriptor.bin")
        .compile(
            &["../../proto/video_conferencing.proto"],
            &["../../proto"],
        )?;
    Ok(())
}
