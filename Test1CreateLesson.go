package main

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/chromedp/chromedp"
	dekanatEvents "github.com/kneu-messenger-pigeon/dekanat-events"
	"github.com/stretchr/testify/assert"
)

func Test1CreateLesson(t *testing.T) {
	fmt.Println("Test1CreateLesson")
	defer printTestResult(t, "Test1CreateLesson")

	err := chooseDiscipline()
	if !assert.NoError(t, err, "Failed to choose discipline") {
		return
	}

	form := findVisibleForm(`.jumbotron form[method="post"]`)
	if !assert.NotNil(t, form, "Form not found") {
		return
	}

	formXPath := form.FullXPathByID()

	ctx, cancel := context.WithTimeout(chromeCtx, time.Second*15)
	defer cancel()

	err = chromedp.Run(ctx, chromedp.Tasks{
		chromedp.Click(formXPath + `//button[text() = "Додати заняття"]`),
		chromedp.WaitVisible(`//h2[text() = "Вид заняття"]`),
	})
	assert.NoError(t, err, "Failed to click on 'Додати заняття' button")

	ctx, cancel = context.WithTimeout(chromeCtx, time.Second*2)
	defer cancel()

	verifyLessonOrScoreForm(t)
	makeScreenshot("create_lesson_form")

	captureScriptUrlReplacer.AssertReplaced(t)

	if t.Failed() {
		return
	}

	ctx, cancel = context.WithTimeout(chromeCtx, time.Second*15)
	defer cancel()

	dekanatReverseProxy.ClearBlockedRequests()
	dekanatReverseProxy.SwitchOffline()
	defer dekanatReverseProxy.SwitchOnline()

	err = chromedp.Run(ctx, chromedp.Tasks{
		chromedp.Click(`//button[contains(text(), "Зберегти")][1]`),
		chromedp.WaitVisible(`//body`),
	})
	assert.NoError(t, err, "Failed to click on 'Зберегти' button")

	expectBlockedPage(t)

	assert.Equal(t, 1, len(dekanatReverseProxy.BlockedRequests), "Wrong number of blocked requests")

	event := realtimeQueue.Fetch(time.Second * 15)

	assert.NotNil(t, event, "Event not found")
	assert.IsType(t, dekanatEvents.LessonCreateEvent{}, event, "Wrong event type")

	lessonCreateEvent, ok := event.(dekanatEvents.LessonCreateEvent)
	if !ok {
		return
	}

	assert.False(t, lessonCreateEvent.HasChanges)
	assert.Equal(t, teacherSession.IsCustomGroup, lessonCreateEvent.IsCustomGroup())

	assert.Equal(t, teacherSession.DisciplineId, lessonCreateEvent.GetDisciplineId(), "Wrong group id")
	assert.Equal(t, teacherSession.Semester, lessonCreateEvent.GetSemester(), "Wrong semester")

	dateNow := time.Now().Format("02.01.2006")
	assert.Equal(t, dateNow, lessonCreateEvent.Date, "Wrong date")

	realtimeQueue.AssertNoOtherEvents(t)

}
