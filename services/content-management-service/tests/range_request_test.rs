//! W10.2 — verify MinIO presigned GET URLs honor HTTP Range requests.
//!
//! Uploads a small object, generates a presigned GET URL via MinioClient, and
//! issues a `Range: bytes=…` request. MinIO must respond with
//! `206 Partial Content`, an `Accept-Ranges: bytes` header, and a
//! `Content-Range` header covering the requested byte slice.
//!
//! Runs only when MinIO test env vars are set. Invoke with:
//!   cargo test --test range_request_test -- --ignored

use bytes::Bytes;
use content_management_service::storage::minio::MinioClient;

fn minio_env() -> Option<(String, String, String, String)> {
    Some((
        std::env::var("TEST_MINIO_ENDPOINT").ok()?,
        std::env::var("TEST_MINIO_ACCESS_KEY").ok()?,
        std::env::var("TEST_MINIO_SECRET_KEY").ok()?,
        std::env::var("TEST_MINIO_BUCKET_PREFIX")
            .unwrap_or_else(|_| "slate-content-test".to_string()),
    ))
}

#[tokio::test]
#[ignore]
async fn presigned_url_returns_206_on_range_request() {
    let (endpoint, access_key, secret_key, prefix) = match minio_env() {
        Some(x) => x,
        None => {
            eprintln!("Skipping: MinIO env vars not set");
            return;
        }
    };

    let minio = MinioClient::new(&endpoint, &access_key, &secret_key, &prefix)
        .await
        .expect("MinioClient::new");

    let bucket = format!("{}-range-test", prefix);
    let key = format!("range-test-{}.bin", uuid::Uuid::new_v4());
    let body: Vec<u8> = (0u8..=255).cycle().take(4096).collect();

    minio.create_bucket(&bucket).await.expect("create_bucket");
    minio
        .put_object(&bucket, &key, Bytes::from(body.clone()), "video/mp4")
        .await
        .expect("put_object");

    let url = minio
        .generate_presigned_download_url(&bucket, &key, 300)
        .await
        .expect("presigned url");

    let client = reqwest::Client::new();
    let resp = client
        .get(&url)
        .header("Range", "bytes=100-199")
        .send()
        .await
        .expect("range request");

    assert_eq!(resp.status().as_u16(), 206, "expected 206 Partial Content");

    let headers = resp.headers();
    let accept_ranges = headers
        .get("accept-ranges")
        .and_then(|v| v.to_str().ok())
        .unwrap_or("");
    assert_eq!(accept_ranges, "bytes", "Accept-Ranges should be 'bytes'");

    let content_range = headers
        .get("content-range")
        .and_then(|v| v.to_str().ok())
        .unwrap_or("")
        .to_string();
    assert!(
        content_range.starts_with("bytes 100-199/"),
        "Content-Range should cover bytes 100-199, got: {}",
        content_range
    );

    let received = resp.bytes().await.expect("body").to_vec();
    assert_eq!(received.len(), 100);
    assert_eq!(received, &body[100..200]);

    // Cleanup best-effort.
    let _ = minio.delete_object(&bucket, &key).await;
}
