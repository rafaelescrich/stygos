# Use Ubuntu as base image
FROM ubuntu:22.04

# Set environment variables to avoid interactive prompts
ENV DEBIAN_FRONTEND=noninteractive

# Install required dependencies
RUN apt-get update && apt-get install -y \
    wget \
    git \
    curl \
    build-essential \
    pkg-config \
    && rm -rf /var/lib/apt/lists/*

# Install Go
RUN wget https://go.dev/dl/go1.21.0.linux-amd64.tar.gz \
    && tar -C /usr/local -xzf go1.21.0.linux-amd64.tar.gz \
    && rm go1.21.0.linux-amd64.tar.gz
ENV PATH=$PATH:/usr/local/go/bin

# Install TinyGo
RUN wget https://github.com/tinygo-org/tinygo/releases/download/v0.30.0/tinygo_0.30.0_amd64.deb \
    && dpkg -i tinygo_0.30.0_amd64.deb \
    && rm tinygo_0.30.0_amd64.deb

# Install Binaryen
RUN wget https://github.com/WebAssembly/binaryen/releases/download/version_116/binaryen-version_116-x86_64-linux.tar.gz \
    && tar -xzf binaryen-version_116-x86_64-linux.tar.gz \
    && mv binaryen-version_116/bin/* /usr/local/bin/ \
    && rm -rf binaryen-version_116*

# Install Brotli
RUN apt-get update && apt-get install -y brotli \
    && rm -rf /var/lib/apt/lists/*

# Install Rust and Cargo Stylus
RUN curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y
ENV PATH="/root/.cargo/bin:${PATH}"
RUN cargo install cargo-stylus

# Set working directory
WORKDIR /app

# Copy project files
COPY . .

# Build the project
RUN go mod download

# Build command
CMD ["make", "build"] 