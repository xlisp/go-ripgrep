package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ANSI 颜色代码
const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorPurple = "\033[35m"
	ColorCyan   = "\033[36m"
	ColorWhite  = "\033[37m"
)

type Config struct {
	fixedStrings bool
	hidden       bool
	noHeading    bool
	lineNumber   bool
	withFilename bool
	color        bool
	ignoreCase   bool
	pattern      string
	searchPath   string
}

func main() {
	var config Config
	
	// 解析命令行参数
	flag.BoolVar(&config.fixedStrings, "fixed-strings", false, "Treat pattern as literal string")
	flag.BoolVar(&config.hidden, "hidden", false, "Search hidden files and directories")
	flag.BoolVar(&config.noHeading, "no-heading", false, "Don't group matches by file")
	flag.BoolVar(&config.lineNumber, "line-number", false, "Show line numbers")
	flag.BoolVar(&config.withFilename, "with-filename", false, "Show filename for each match")
	flag.StringVar(&config.pattern, "pattern", "", "Search pattern")
	flag.BoolVar(&config.ignoreCase, "ignore-case", false, "Case insensitive search")
	
	// 自定义color参数处理
	colorFlag := flag.String("color", "never", "When to use colors (never, always, auto)")
	
	flag.Parse()
	
	// 处理color参数
	config.color = *colorFlag == "always" || (*colorFlag == "auto" && isTerminal())
	
	// 获取剩余参数 (pattern 和 path)
	args := flag.Args()
	if len(args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] -- pattern path\n", os.Args[0])
		os.Exit(1)
	}
	
	config.pattern = args[0]
	config.searchPath = args[1]
	
	// 执行搜索
	err := search(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func search(config Config) error {
	return filepath.Walk(config.searchPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // 忽略错误，继续搜索
		}
		
		// 跳过目录
		if info.IsDir() {
			// 如果不搜索隐藏文件，跳过隐藏目录
			if !config.hidden && isHidden(path) {
				return filepath.SkipDir
			}
			return nil
		}
		
		// 如果不搜索隐藏文件，跳过隐藏文件
		if !config.hidden && isHidden(path) {
			return nil
		}
		
		// 跳过一些明显的二进制文件类型
		if isBinaryFileByExtension(path) {
			return nil
		}
		
		// 搜索文件内容
		return searchInFile(path, config)
	})
}

func searchInFile(filename string, config Config) error {
	file, err := os.Open(filename)
	if err != nil {
		return nil // 忽略无法打开的文件
	}
	defer file.Close()
	
	// 检查文件是否为二进制文件
	if isBinaryFile(file) {
		return nil
	}
	
	// 重置文件指针
	file.Seek(0, 0)
	
	scanner := bufio.NewScanner(file)
	
	// 增加缓冲区大小来处理长行
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 10*1024*1024) // 最大10MB的行
	
	lineNum := 0
	
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		
		// 限制行长度显示，避免终端显示问题
		if len(line) > 32768 {
			line = line[:32768] + "... [line truncated]"
		}
		
		if matchesPattern(line, config.pattern, config.fixedStrings, config.ignoreCase) {
			printMatch(filename, lineNum, line, config)
		}
	}
	
	return scanner.Err()
}

func matchesPattern(line, pattern string, fixedStrings, ignoreCase bool) bool {
	if ignoreCase {
		line = strings.ToLower(line)
		pattern = strings.ToLower(pattern)
	}
	
	if fixedStrings {
		return strings.Contains(line, pattern)
	}
	
	// 简单的字符串匹配（这里可以扩展为正则表达式）
	return strings.Contains(line, pattern)
}

func printMatch(filename string, lineNum int, line string, config Config) {
	var parts []string
	
	// 添加文件名
	if config.withFilename {
		if config.color {
			parts = append(parts, ColorPurple+filename+ColorReset)
		} else {
			parts = append(parts, filename)
		}
	}
	
	// 添加行号
	if config.lineNumber {
		if config.color {
			parts = append(parts, ColorGreen+fmt.Sprintf("%d", lineNum)+ColorReset)
		} else {
			parts = append(parts, fmt.Sprintf("%d", lineNum))
		}
	}
	
	// 高亮匹配的文本
	displayLine := line
	if config.color {
		displayLine = highlightMatches(line, config.pattern, config.fixedStrings, config.ignoreCase)
	}
	
	// 组合输出
	if len(parts) > 0 {
		fmt.Printf("%s:%s\n", strings.Join(parts, ":"), displayLine)
	} else {
		fmt.Println(displayLine)
	}
}

func highlightMatches(line, pattern string, fixedStrings, ignoreCase bool) string {
	if len(pattern) == 0 {
		return line
	}
	
	// 使用更安全的高亮方法
	if ignoreCase {
		return highlightIgnoreCase(line, pattern)
	}
	
	// 大小写敏感的简单替换
	return strings.ReplaceAll(line, pattern, ColorRed+pattern+ColorReset)
}

// 安全的忽略大小写高亮函数
func highlightIgnoreCase(line, pattern string) string {
	if len(line) == 0 || len(pattern) == 0 {
		return line
	}
	
	lowerLine := strings.ToLower(line)
	lowerPattern := strings.ToLower(pattern)
	
	var result strings.Builder
	lastIndex := 0
	
	for {
		index := strings.Index(lowerLine[lastIndex:], lowerPattern)
		if index == -1 {
			// 添加剩余部分
			result.WriteString(line[lastIndex:])
			break
		}
		
		actualIndex := lastIndex + index
		
		// 检查边界
		if actualIndex+len(pattern) > len(line) {
			result.WriteString(line[lastIndex:])
			break
		}
		
		// 添加匹配前的部分
		result.WriteString(line[lastIndex:actualIndex])
		
		// 添加高亮的匹配部分
		original := line[actualIndex : actualIndex+len(pattern)]
		result.WriteString(ColorRed + original + ColorReset)
		
		lastIndex = actualIndex + len(pattern)
	}
	
	return result.String()
}

func isHidden(path string) bool {
	name := filepath.Base(path)
	return strings.HasPrefix(name, ".")
}

func isTerminal() bool {
	// 简单检查是否为终端
	stat, _ := os.Stdout.Stat()
	return (stat.Mode() & os.ModeCharDevice) != 0
}

// 检查文件是否为二进制文件
func isBinaryFile(file *os.File) bool {
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil {
		return false
	}
	
	// 检查前512字节中是否包含空字节
	for i := 0; i < n; i++ {
		if buffer[i] == 0 {
			return true
		}
	}
	return false
}

// 根据文件扩展名判断是否为二进制文件
func isBinaryFileByExtension(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	binaryExts := []string{
		".exe", ".dll", ".so", ".dylib", ".a", ".o", ".obj",
		".zip", ".tar", ".gz", ".bz2", ".xz", ".7z", ".rar",
		".jpg", ".jpeg", ".png", ".gif", ".bmp", ".tiff", ".ico",
		".mp3", ".mp4", ".avi", ".mkv", ".mov", ".wmv", ".flv",
		".pdf", ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx",
		".bin", ".dat", ".db", ".sqlite", ".sqlite3",
		".pyc", ".class", ".jar",
	}
	
	for _, binExt := range binaryExts {
		if ext == binExt {
			return true
		}
	}
	return false
}
