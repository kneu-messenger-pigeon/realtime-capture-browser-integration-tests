package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"

	"github.com/chromedp/chromedp"
)

func createChromeContext(chromeWsUrl string) (context.Context, context.CancelFunc) {
	var allocCtx context.Context
	var allocCtxCancel context.CancelFunc

	if chromeWsUrl == "EXEC" || chromeWsUrl == "DESKTOP" {
		allocCtx, allocCtxCancel = createDesktopChromeAllocator(chromeWsUrl != "DESKTOP")
	} else {
		allocCtx, allocCtxCancel = createRemoteChromeAllocator(chromeWsUrl)
	}

	logFile, err := os.Create("chrome.log")
	if err != nil {
		log.Fatal(err)
	}

	var removeBase64Data = regexp.MustCompile(`"data":"/9j/.*?"`)
	logPrint := func(format string, v ...any) {
		logRecord := fmt.Sprintf(format, v...)
		logRecord = removeBase64Data.ReplaceAllString(logRecord, `"data":"<base64 data>"`)
		_, _ = fmt.Fprintln(logFile, logRecord)
	}

	taskCtx, cancel := chromedp.NewContext(allocCtx, chromedp.WithLogf(logPrint), chromedp.WithDebugf(logPrint))

	return taskCtx, func() {
		cancel()
		allocCtxCancel()

		_ = logFile.Close()
	}
}

func createRemoteChromeAllocator(chromeWsUrl string) (context.Context, context.CancelFunc) {
	devtoolsWsURL := flag.String("devtools-ws-url", chromeWsUrl, "DevTools WebSocket URL")
	flag.Parse()

	return chromedp.NewRemoteAllocator(context.Background(), *devtoolsWsURL)
}

func createDesktopChromeAllocator(headless bool) (context.Context, context.CancelFunc) {
	return chromedp.NewExecAllocator(
		context.Background(),
		append(chromedp.DefaultExecAllocatorOptions[:], chromedp.Flag("headless", headless))...,
	)
}
