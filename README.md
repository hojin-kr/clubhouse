# Concept
Common app server for
Google Cloud Platform (CloudRun)
& Golang
& GRPC
& Protobuff

# Feature
- GCP
- CloudRun
- Datastore only
- GRPC & Protobuff
- Container
- Github & Github Actions CI/CD
- Apns (Apple iOS Server Push)
- Low Cost

---
# CI/CD Flow

1. Github : code push
2. Github Actions : Build submit to GCP Cloud Build (This sequentially builds and pushes containers by GCP)
4. GCP CloudRun : Run containers and deploy services