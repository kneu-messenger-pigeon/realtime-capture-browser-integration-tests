package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/stretchr/testify/assert"
)

var config Config

var stubMatch = func(pat, str string) (bool, error) { return true, nil }

var chromeCtx context.Context

var captureScriptUrlReplacer *CaptureScriptUrlReplacer

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

	captureScriptUrlReplacer = NewCaptureScriptUrlReplacer(config.scriptProdPublicUrl, config.scriptPublicUrl)

	dekanatReverseProxy = NewReverseProxy(config.dekenatWebHost, captureScriptUrlReplacer.Replace)

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

	fmt.Printf("::add-mask::%s\n", teacherWithActiveLesson.Password)
	fmt.Printf("Teacher with active lesson: %+v\n", teacherWithActiveLesson)

	teacherSession = NewTeacherSession(teacherWithActiveLesson)

	test := testing.InternalTest{
		Name: "integration testing",
		F: func(t *testing.T) {
			realtimeQueue = CreateRealtimeQueue(t)
			if realtimeQueue == nil {
				t.Fatal("Failed to create realtime queue")
				return
			}

			reverseProxyTestPass := t.Run("TestReverseProxy", TestReverseProxy)
			if !reverseProxyTestPass {
				t.Fatal("TestReverseProxy failed")
				return
			}

			err = chromedp.Run(chromeCtx, chromedp.EmulateViewport(1280, 1024))
			assert.NoError(t, err)

			t.Run("RegularGroup", TestRegularGroup)
			t.Run("CustomGroup", TestCustomGroup)

			if !t.Failed() && config.chromeWsUrl == "DESKTOP" {
				fmt.Print("Press enter to exit")
				_, _ = bufio.NewReader(os.Stdin).ReadBytes('\n')
			}
		},
	}

	testing.Main(stubMatch, []testing.InternalTest{test}, []testing.InternalBenchmark{}, []testing.InternalExample{})

}

func TestRegularGroup(t *testing.T) {
	assert.False(t, teacherSession.IsCustomGroup)

	logoutFunc := LoginAndFetchGroupPageUrl(t, teacherSession)
	defer logoutFunc()

	fmt.Println("Regular group: Start testing in regular group " + time.Now().Format("2006-01-02-15-04-05"))
	setupTests(t)
	fmt.Println("Regular group: test group done")
}

func TestCustomGroup(t *testing.T) {
	teacherWithActiveLesson := dekanatRepository.GetTeacherWithActiveLessonInCustomGroup()
	if teacherWithActiveLesson == nil {
		t.Skip("Teacher with active lesson in custom group not found. Skip test")
		return
	}

	fmt.Printf("::add-mask::%s\n", teacherWithActiveLesson.Password)
	fmt.Printf("Teacher with active lesson in custom group: %+v\n", teacherWithActiveLesson)

	teacherSession = NewTeacherSession(teacherWithActiveLesson)
	if !assert.True(t, teacherSession.IsCustomGroup) {
		return
	}

	logoutFunc := LoginAndFetchGroupPageUrl(t, teacherSession)
	defer logoutFunc()

	fmt.Println("Custom group: Start testing " + time.Now().Format("2006-01-02-15-04-05"))
	setupTests(t)
	fmt.Println("Custom group: Test done")
}
