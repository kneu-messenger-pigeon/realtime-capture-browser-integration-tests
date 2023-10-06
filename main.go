package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"github.com/chromedp/chromedp"
	"github.com/stretchr/testify/assert"
	"log"
	"os"
	"testing"
)

var config Config

var stubMatch = func(pat, str string) (bool, error) { return true, nil }

var chromeCtx context.Context

var dekanatRepository *DekanatRepository

var dekanatReverseProxy *DekanatReverseProxy

var teacherSession = &TeacherSession{}

var realtimeQueue = &RealtimeQueue{}

func main() {
	var err error
	var cancel context.CancelFunc

	envFilename := ""
	if _, err = os.Stat(".env"); err == nil {
		envFilename = ".env"
	}

	config, err = loadConfig(envFilename)

	dekanatReverseProxy = NewReverseProxy(config.dekenatWebHost, func(body []byte) []byte {
		return bytes.ReplaceAll(body, config.scriptProdPublicUrl, config.scriptPublicUrl)
	})

	// create context
	chromeCtx, cancel = createChromeContext(config.chromeWsUrl)
	defer cancel()

	dekanatRepository, err = NewDekanatRepository(config.dekanatDbDSN, config.dekanatSecret)
	if err != nil {
		log.Fatal(err)
		return
	}
	defer dekanatRepository.Close()

	teacherWithActiveLesson := dekanatRepository.GetTeacherWithActiveLesson()
	if teacherWithActiveLesson == nil {
		log.Fatal("Teacher with active lesson not found")
	}

	fmt.Printf("Teacher with active lesson: %+v\n", teacherWithActiveLesson)

	teacherSession = NewTeacherSession(teacherWithActiveLesson)

	test := testing.InternalTest{
		Name: "integration testing",
		F: func(t *testing.T) {
			reverseProxyTestPass := true || t.Run("TestReverseProxy", TestReverseProxy)
			if !reverseProxyTestPass {
				t.Fatal("TestReverseProxy failed")
				return
			}

			realtimeQueue = CreateRealtimeQueue(t)

			err = chromedp.Run(chromeCtx)
			assert.NoError(t, err)

			teacherSession.GroupPageUrl = `http://mbp-anton:8090/cgi-bin/teachers.cgi?sesID=23BBEBB0-F3E6-4822-AEC7-026FF57DC112&n=1&grp=%D0%CC-306&teacher=321`
			if teacherSession.GroupPageUrl == "" {
				LoginAndFetchGroupPageUrl(t, teacherSession)
			}

			fmt.Println("Start testing..")
			setupTests(t)
			fmt.Println("Test done")

			fmt.Print("Press enter to exit")

			if !t.Failed() {
				_, _ = bufio.NewReader(os.Stdin).ReadBytes('\n')
			}
		},
	}

	testing.Main(stubMatch, []testing.InternalTest{test}, []testing.InternalBenchmark{}, []testing.InternalExample{})

}
