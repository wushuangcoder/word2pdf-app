FROM registry.cn-beijing.aliyuncs.com/usy/libreoffice:25.8.1

# ==============================
# 1. 更换 apk 源为阿里云镜像
# ==============================
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories

# ==============================
# 2. 安装中文字体和依赖（核心！）
# ==============================
RUN apk update && \
    apk add --no-cache \
        # 字体配置工具
        fontconfig \
        # 支持更多 CJK 字符
        font-noto-cjk && \
    # 重建字体缓存
    fc-cache -fv

# ==============================
# 3. 安装并配置 locale
# ==============================
RUN apk add --no-cache \
        tzdata \
        musl-locales \
        musl-locales-lang && \
    echo "LANG=zh_CN.UTF-8" > /etc/profile.d/locale.sh && \
    echo "LC_ALL=zh_CN.UTF-8" >> /etc/profile.d/locale.sh

# ==============================
# 4. 设置默认时区为上海
# ==============================
RUN ln -sf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo "Asia/Shanghai" > /etc/timezone

# ==============================
# 5. 安装 Go 环境 (版本 1.25)
# ==============================
RUN apk add --no-cache \
        git \
        build-base && \
    # 下载并安装 Go 1.25
    wget -O /tmp/go1.25.0.linux-amd64.tar.gz https://go.dev/dl/go1.25.0.linux-amd64.tar.gz && \
    rm -rf /usr/local/go && \
    tar -C /usr/local -xzf /tmp/go1.25.0.linux-amd64.tar.gz && \
    rm /tmp/go1.25.0.linux-amd64.tar.gz

# 设置 Go 环境变量
ENV PATH=$PATH:/usr/local/go/bin
ENV GOPATH=/go
ENV GOBIN=/go/bin

# ==============================
# 6. 复制并构建 Go 应用
# ==============================
WORKDIR /app

# 设置 Go 代理
ENV GOPROXY=https://goproxy.cn,direct

# 复制 go.mod 和 go.sum
COPY go.mod go.sum .

# 下载依赖
RUN go get github.com/gin-gonic/gin

# 复制应用代码
COPY main.go .

# 构建应用
RUN go build -o word2pdf-server .

# ==============================
# 7. （可选）验证配置
# ==============================
RUN echo "✅ Font & Locale Setup Complete" && \
    fc-list :lang=zh | head -10 && \
    date

# ==============================
# 8. 启动应用
# ==============================
EXPOSE 8080

CMD ["/app/word2pdf-server"]