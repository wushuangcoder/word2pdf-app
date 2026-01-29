package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

func main() {
	// 设置日志输出
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("正在启动Word2PDF服务...")

	// 设置Gin模式
	gin.SetMode(gin.ReleaseMode)

	// 创建Gin引擎
	r := gin.Default()

	// 添加CORS中间件
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	})

	// 健康检查接口
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"message": "Word2PDF service is running",
		})
	})

	// 文档转PDF接口
	r.POST("/convert", func(c *gin.Context) {
		log.Println("接收到文件转换请求")
		// 获取上传的文件
		file, header, err := c.Request.FormFile("file")
		if err != nil {
			log.Printf("接收文件失败: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "未上传文件",
			})
			return
		}
		log.Printf("接收到文件: %s, 大小: %d 字节", header.Filename, header.Size)
		defer file.Close()

		// 获取文件扩展名
		ext := strings.ToLower(filepath.Ext(header.Filename))

		// 检查文件类型是否支持
		supportedTypes := map[string]bool{
			".doc":  true,
			".docx": true,
			".xls":  true,
			".xlsx": true,
			".ppt":  true,
			".pptx": true,
		}

		log.Printf("检查文件类型: %s", ext)
		if !supportedTypes[ext] {
			log.Printf("不支持的文件类型: %s", ext)
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "不支持的文件类型。请上传Word、Excel或PowerPoint文件。",
			})
			return
		}
		log.Println("文件类型支持")

		// 创建临时文件
		log.Println("正在创建临时目录...")
		tempDir, err := os.MkdirTemp("", "word2pdf")
		if err != nil {
			log.Printf("创建临时目录失败: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "创建临时目录失败",
			})
			return
		}
		log.Printf("创建临时目录成功: %s", tempDir)
		defer os.RemoveAll(tempDir)

		// 保存上传的文件
		inputPath := filepath.Join(tempDir, header.Filename)
		outputPath := filepath.Join(tempDir, strings.TrimSuffix(header.Filename, ext)+"_output.pdf")
		log.Printf("输入文件路径: %s", inputPath)
		log.Printf("输出文件路径: %s", outputPath)

		// 创建输出文件
		log.Println("正在创建临时文件...")
		dst, err := os.Create(inputPath)
		if err != nil {
			log.Printf("创建临时文件失败: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "创建临时文件失败",
			})
			return
		}
		defer dst.Close()

		// 复制文件内容
		log.Println("正在保存上传文件...")
		if _, err = io.Copy(dst, file); err != nil {
			log.Printf("保存上传文件失败: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "保存上传文件失败",
			})
			return
		}
		log.Println("文件保存成功")

		// 使用LibreOffice将文档转换为PDF
		log.Println("正在使用LibreOffice将文件转换为PDF...")
		cmd := exec.Command("libreoffice", "--headless", "--convert-to", "pdf", "--outdir", tempDir, inputPath)
		log.Printf("执行命令: %v", cmd.Args)
		output, err := cmd.CombinedOutput()
		if err != nil {
			log.Printf("文件转换为PDF失败: %v", err)
			log.Printf("LibreOffice输出: %s", string(output))
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "文件转换为PDF失败",
				"details": string(output),
			})
			return
		}
		log.Println("文件转换为PDF成功")
		if len(output) > 0 {
			log.Printf("LibreOffice输出: %s", string(output))
		}

		// 检查转换后的文件是否存在
		log.Printf("检查转换后的PDF文件是否存在: %s", outputPath)
		if _, err := os.Stat(outputPath); os.IsNotExist(err) {
			log.Printf("未找到转换后的PDF文件: %s", outputPath)
			// 尝试使用不同的输出路径格式
			outputPath = filepath.Join(tempDir, strings.TrimSuffix(filepath.Base(header.Filename), ext)+".pdf")
			log.Printf("尝试使用替代路径: %s", outputPath)
			if _, err := os.Stat(outputPath); os.IsNotExist(err) {
				log.Printf("在替代路径中未找到转换后的PDF文件: %s", outputPath)
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "未找到转换后的PDF文件",
				})
				return
			}
			log.Printf("在替代路径中找到转换后的PDF文件: %s", outputPath)
		} else {
			log.Printf("找到转换后的PDF文件: %s", outputPath)
		}

		// 读取转换后的PDF文件
		log.Println("正在打开转换后的PDF文件...")
		pdfFile, err := os.Open(outputPath)
		if err != nil {
			log.Printf("打开转换后的PDF文件失败: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "打开转换后的PDF文件失败",
			})
			return
		}
		defer pdfFile.Close()

		// 设置响应头
		log.Println("正在设置响应头...")
		c.Header("Content-Description", "File Transfer")
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", strings.TrimSuffix(header.Filename, ext)+".pdf"))
		c.Header("Content-Type", "application/pdf")

		// 将PDF文件写入响应
		log.Println("正在向客户端发送PDF文件...")
		if _, err := io.Copy(c.Writer, pdfFile); err != nil {
			log.Printf("发送PDF文件失败: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "发送PDF文件失败",
			})
			return
		}
		log.Println("PDF文件发送成功")
	})

	// 启动服务器
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		log.Println("未设置PORT环境变量，使用默认端口8080")
	}

	log.Printf("服务器正在端口%s上启动...", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("服务器启动失败: %v", err)
	}
}
