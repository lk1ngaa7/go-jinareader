package main

import (
	"encoding/json"
	"fmt"
	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/JohannesKaufmann/html-to-markdown/plugin"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/go-shiori/go-readability"
	"github.com/sirupsen/logrus"
)

var log = logrus.New()

func init() {
	log.SetOutput(os.Stdout)
	log.SetLevel(logrus.InfoLevel)
	log.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})
}

type Response struct {
	Markdown string `json:"markdown"`
	Error    string `json:"error,omitempty"`
}

func fetchWebpage(url string) (string, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("User-Agent", "Golang WebpageToMarkdown Bot/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error fetching webpage: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response body: %w", err)
	}

	return string(body), nil
}

func extractContent(html string) (string, error) {
	article, err := readability.FromReader(strings.NewReader(html), nil)
	if err != nil {
		return "", fmt.Errorf("error extracting content: %w", err)
	}
	fmt.Printf("article.Title %v\n", article.Title)
	fmt.Printf("article.Byline %v \n", article.Byline)
	fmt.Printf("article.Image %v \n", article.Image)
	fmt.Printf("article.Published %v \n", article.PublishedTime)
	fmt.Printf("article.Content %v \n", article.Content)
	fmt.Printf("article.Excerpt %v \n", article.Excerpt)
	fmt.Printf("article.URL %v \n", article.Node)
	fmt.Printf("article.TextContent %v \n", article.TextContent)
	return article.Content, nil
}

func getConverter(domain string) *md.Converter {
	// 创建一个全面配置的 HTML 到 Markdown 转换器
	converter := md.NewConverter(domain, true, &md.Options{
		// 基本样式选项
		HeadingStyle:     "atx",    // 可选: "setext"
		HorizontalRule:   "---",    // 可选: "***", "___"
		BulletListMarker: "-",      // 可选: "*", "+"
		CodeBlockStyle:   "fenced", // 可选: "indented"
		// 链接相关选项
		LinkStyle:          "inlined", // 可选: "referenced"
		LinkReferenceStyle: "full",    // 可选: "collapsed", "shortcut"
		// 强调和删除线选项
		StrongDelimiter: "**", // 可选: "__"
		EmDelimiter:     "_",  // 可选: "*"
	})
	// 添加插件
	converter.Use(plugin.GitHubFlavored())
	converter.Use(plugin.Table())
	converter.Use(plugin.Strikethrough(""))
	converter.Use(plugin.YoutubeEmbed())
	converter.Use(plugin.TaskListItems())
	converter.Use(plugin.VimeoEmbed(60))
	return converter
}
func convertToMarkdown(content string, domain string) string {
	// 创建 HTML 到 Markdown 的转换器
	converter := getConverter(domain)

	// 转换 HTML 到 Markdown
	markdown, err := converter.ConvertString(content)
	if err != nil {
		log.Fatalf("Error converting to Markdown: %v", err)
	}
	return markdown
}

func handleWebpageToMarkdown(w http.ResponseWriter, r *http.Request) {
	fmt.Println("handleWebpageToMarkdown")
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var requestBody struct {
		URL string `json:"url"`
	}

	err := json.NewDecoder(r.Body).Decode(&requestBody)
	fmt.Println(requestBody.URL)
	if err != nil {
		log.WithError(err).Error("Failed to decode request body")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if requestBody.URL == "" {
		http.Error(w, "URL is required", http.StatusBadRequest)
		return
	}

	log.WithField("url", requestBody.URL).Info("Processing request")

	html, err := fetchWebpage(requestBody.URL)
	if err != nil {
		log.WithError(err).WithField("url", requestBody.URL).Error("Failed to fetch webpage")
		sendJSONResponse(w, Response{Error: fmt.Sprintf("Failed to fetch webpage: %v", err)}, http.StatusInternalServerError)
		return
	}

	content, _ := extractContent(html)
	if err != nil {
		log.WithError(err).WithField("url", requestBody.URL).Error("Failed to extract content")
		sendJSONResponse(w, Response{Error: fmt.Sprintf("Failed to extract content: %v", err)}, http.StatusInternalServerError)
		return
	}
	//url获取domain
	domain := getDomainFromURL(requestBody.URL)
	markdown := convertToMarkdown(content, domain)

	log.WithField("url", requestBody.URL).Info("Successfully processed webpage")
	fmt.Println(markdown)
	sendJSONResponse(w, Response{Markdown: markdown}, http.StatusOK)
}
func getDomainFromURL(urlStr string) string {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		log.WithError(err).WithField("url", urlStr).Error("Failed to parse URL")
		return ""
	}
	return parsedURL.Host
}
func sendJSONResponse(w http.ResponseWriter, response Response, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8087"
	}

	http.HandleFunc("/convert", handleWebpageToMarkdown)
	fmt.Println("Server listening on port " + port)
	log.Infof("Server starting on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.WithError(err).Fatal("Server failed to start")
	}
}
