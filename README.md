# AR-Code

AR-Code is a browser-based Augmented Reality platform that allows businesses and sellers to showcase products in AR using QR codes.

Built with a CDN-backed architecture and Redis caching, the platform delivers fast-loading 3D models that can be launched instantly on both iOS and Android without installing any application.
<br>Frontend Codebase: https://github.com/ishan-xy/arcode-dashboard
## Preview

### Home Page

<p align="center">
  <img 
    src="https://github.com/user-attachments/assets/f03baf36-9b72-47cc-8315-15d4b04f46e7"
    alt="Home Page"
    width="900"
  />
</p>

### Guest Upload

<p align="center">
  <img 
    src="https://github.com/user-attachments/assets/36005651-06f3-4254-a6d4-1bfa6a815529"
    alt="Guest Upload"
    width="900"
  />
</p>

### Logged In User Upload

<p align="center">
  <img 
    src="https://github.com/user-attachments/assets/809de82a-7487-4bd6-9aba-d6cc9ac11d9f"
    alt="Logged In Upload"
    width="900"
  />
</p>

### AR Product View

<p align="center">
  <img 
    src="https://github.com/user-attachments/assets/1c7cba81-34ab-42dd-b9b9-bd494025a80c"
    alt="AR Product Preview"
    width="300"
  />
</p>

---

## Features

- Browser-based AR experience using QR codes
- Instant 3D model previews on mobile devices
- App-free support for both iOS and Android
- CDN-backed asset delivery for faster model loading
- Redis caching for pre-signed CDN URLs
- Guest uploads and authenticated dashboards
- Secure and scalable backend architecture

---

## Tech Stack

- Go
- Fiber
- SolidJS
- Docker
- Nginx
- Redis

---

## How It Works

1. Upload a 3D model
2. Generate a QR code
3. Scan the QR code on a mobile device
4. Instantly launch the model in AR

---

## Performance

- Reduced signed-link generation latency by 95% using Redis caching
- Optimized CDN delivery for fast 3D asset access
- Lightweight deployment architecture using Docker and Nginx

---

## Use Cases

- E-commerce product visualization
- Furniture and decor previews
- Product marketing and packaging
- Retail and showroom experiences

---

## License

MIT License
