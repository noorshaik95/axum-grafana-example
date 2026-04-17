use tokio::io::AsyncWriteExt;
use tokio::net::TcpListener;

/// Starts a minimal HTTP health server on `addr` (e.g. "0.0.0.0:9081").
/// Responds to every TCP connection with a fixed HTTP 200 JSON response,
/// which is sufficient for Prometheus up/down scraping and Docker HEALTHCHECK.
pub async fn start_health_server(addr: &str) -> anyhow::Result<()> {
    let listener = TcpListener::bind(addr).await?;
    tracing::info!("Health server listening on {}", addr);

    loop {
        match listener.accept().await {
            Ok((mut stream, _peer)) => {
                tokio::spawn(async move {
                    let response = concat!(
                        "HTTP/1.1 200 OK\r\n",
                        "Content-Type: application/json\r\n",
                        "Content-Length: 27\r\n",
                        "Connection: close\r\n",
                        "\r\n",
                        r#"{"status":"ok","service":"#,
                        r#""onboarding-service"}"#,
                    );
                    let _ = stream.write_all(response.as_bytes()).await;
                });
            }
            Err(e) => {
                tracing::warn!("Health server accept error: {}", e);
            }
        }
    }
}
