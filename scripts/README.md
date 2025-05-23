`build-multiplatform.sh`

## 🚀 **Key Features**

### **Platform Support**

- **Linux**: amd64, arm64, 386
- **Windows**: amd64, 386
- **macOS**: amd64, arm64 (Intel & Apple Silicon)
- **FreeBSD**: amd64, 386

### **Advanced Build Features**

- **Version Injection**: Git tags, commit hash, build time
- **Binary Compression**: UPX integration for smaller binaries
- **Archive Creation**: ZIP (Windows) / TAR.GZ (Unix)
- **Docker Multi-arch**: Cross-platform container builds
- **Checksums**: SHA256 for integrity verification
- **Build Reports**: Detailed build statistics

### **Command Line Options**

bash

```bash
# Basic usage
./scripts/build-multiplatform.sh

# Advanced usage
./scripts/build-multiplatform.sh -c -z -a -d
# -c: Clean previous builds
# -z: Compress with UPX  
# -a: Create archives
# -d: Build Docker multi-arch images

# Custom version and output
./scripts/build-multiplatform.sh -v v1.2.3 -o releases
```

## 📁 **Output Structure**

```
dist/
├── ip-marketplace-backend-linux-amd64
├── ip-marketplace-backend-linux-amd64.sha256
├── ip-marketplace-backend-windows-amd64.exe
├── ip-marketplace-backend-darwin-arm64
├── ip-marketplace-backend-v1.2.3-linux-amd64.tar.gz
├── ip-marketplace-backend-v1.2.3-windows-amd64.zip
└── build-report.txt
```

## 🔧 **Usage Examples**

### **Development Build**

bash

```bash
chmod +x scripts/build-multiplatform.sh
./scripts/build-multiplatform.sh
```

### **Production Release**

bash

```bash
./scripts/build-multiplatform.sh \
  --clean \
  --compress \
  --archive \
  --version v1.0.0
```

### **Docker Multi-arch**

bash

```bash
export DOCKER_REGISTRY=your-registry.com
./scripts/build-multiplatform.sh --docker --push
```
