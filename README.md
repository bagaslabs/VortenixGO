## About VortenixGo

VortenixGo is a Growtopia multibot program designed to manage and control multiple bots simultaneously using a modern and flexible architecture. The project is built with a strong separation between backend logic and user interface, allowing it to remain lightweight, scalable, and easy to extend. Instead of relying on a native GUI, VortenixGo uses a web-based interface to provide real-time control and monitoring.

The main user interface of VortenixGo is implemented using WebSocket technology and runs locally on `http://localhost:8080`. This approach allows the UI to be accessed directly from a web browser, making it usable on both desktop and mobile devices without additional client software. WebSocket enables real-time, bidirectional communication between the Go backend and the browser-based frontend, ensuring fast updates for bot status, logs, and actions.

VortenixGo operates using two primary network layers. The first layer is HTTPS, which is responsible for handling secure communication such as account authentication, token validation, and interactions with Growtopia’s official services. This layer follows standard HTTPS communication practices to ensure reliability and security during sensitive operations.

The second network layer is the UDO network, which is used for the core Growtopia gameplay connection. This layer is built on top of ENet written in C++, which has been modified specifically to match Growtopia’s networking behavior and protocol. The ENet implementation used in VortenixGo has been adapted to properly handle Growtopia’s packet structure, connection timing, channel behavior, and reliability requirements. These modifications are essential for stable and accurate communication with Growtopia servers.

For reference and further development, similar Growtopia-compatible ENet implementations can be found on GitHub under the user Cloei, particularly in repositories such as Rusty ENet and Mori. These repositories provide insight into how ENet can be customized to align with Growtopia’s network protocol and are commonly used as a base or reference in Growtopia-related networking projects.

Overall, VortenixGo aims to provide a clean, modular, and extensible multibot system by combining a Go-based backend, a WebSocket-driven web UI, and a customized ENet networking layer for Growtopia connectivity. The project is intended for experimental and educational purposes, focusing on architecture, networking, and system design rather than traditional native GUI approaches.

## 📦 System Requirement 

- **Go (Golang) 64-bit**
  - Windows 64-bit
  - Linux 64-bit
- Terminal / Command Prompt

> ⚠️ **Note:**  
> Go version 32-bit **not supported**.

