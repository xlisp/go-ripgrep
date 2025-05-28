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
	
	scanner := bufio.NewScanner(file)
	lineNum := 0
	
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		
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
	var output strings.Builder
	
	// 构建输出格式
	if config.withFilename {
		if config.color {
			output.WriteString(ColorPurple + filename + ColorReset)
		} else {
			output.WriteString(filename)
		}
		output.WriteString(":")
	}
	
	if config.lineNumber {
		if config.color {
			output.WriteString(ColorGreen + fmt.Sprintf("%d", lineNum) + ColorReset)
		} else {
			output.WriteString(fmt.Sprintf("%d", lineNum))
		}
		output.WriteString(":")
	}
	
	// 高亮匹配的文本
	if config.color {
		highlightedLine := highlightMatches(line, config.pattern, config.fixedStrings, config.ignoreCase)
		output.WriteString(highlightedLine)
	} else {
		output.WriteString(line)
	}
	
	fmt.Println(output.String())
}

func highlightMatches(line, pattern string, fixedStrings, ignoreCase bool) string {
	if !fixedStrings {
		// 简单高亮，实际ripgrep会更复杂
		searchPattern := pattern
		if ignoreCase {
			// 找到原始大小写的匹配
			lowerLine := strings.ToLower(line)
			lowerPattern := strings.ToLower(pattern)
			
			result := line
			index := 0
			for {
				pos := strings.Index(lowerLine[index:], lowerPattern)
				if pos == -1 {
					break
				}
				actualPos := index + pos
				original := line[actualPos : actualPos+len(pattern)]
				highlighted := ColorRed + original + ColorReset
				result = result[:actualPos] + highlighted + result[actualPos+len(pattern):]
				index = actualPos + len(highlighted)
				lowerLine = strings.ToLower(result)
			}
			return result
		} else {
			return strings.ReplaceAll(line, searchPattern, ColorRed+searchPattern+ColorReset)
		}
	}
	
	// fixed-strings 模式
	if ignoreCase {
		lowerLine := strings.ToLower(line)
		lowerPattern := strings.ToLower(pattern)
		
		result := line
		index := 0
		for {
			pos := strings.Index(lowerLine[index:], lowerPattern)
			if pos == -1 {
				break
			}
			actualPos := index + pos
			original := line[actualPos : actualPos+len(pattern)]
			highlighted := ColorRed + original + ColorReset
			result = result[:actualPos] + highlighted + result[actualPos+len(pattern):]
			index = actualPos + len(highlighted)
			lowerLine = strings.ToLower(result)
		}
		return result
	}
	
	return strings.ReplaceAll(line, pattern, ColorRed+pattern+ColorReset)
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
