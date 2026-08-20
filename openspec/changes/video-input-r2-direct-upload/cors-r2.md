# R2 CORS for browser direct upload

The video tool uploads reference media with a browser `PUT` to a short-lived R2 presigned URL. The R2 bucket **must** allow CORS from the dashboard origin(s).

Example CORS rules (adjust origins to your deployment):

```json
[
  {
    "AllowedOrigins": ["https://your-dashboard.example.com"],
    "AllowedMethods": ["PUT", "GET", "HEAD", "OPTIONS"],
    "AllowedHeaders": ["*"],
    "ExposeHeaders": ["ETag"],
    "MaxAgeSeconds": 3600
  }
]
```

Without PUT/OPTIONS from the dashboard origin, the browser upload step fails even when presign succeeds.
