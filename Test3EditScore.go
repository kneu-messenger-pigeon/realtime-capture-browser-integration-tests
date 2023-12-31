package main

import (
	"context"
	"fmt"
	"github.com/chromedp/chromedp"
	dekanatEvents "github.com/kneu-messenger-pigeon/dekanat-events"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestTest3EditScore(t *testing.T) {
	fmt.Println("TestTest3EditScore")
	defer printTestResult(t, "TestTest3EditScore")

	err := chooseDiscipline()
	if !assert.NoError(t, err, "Failed to choose discipline") {
		return
	}

	err = openLessonPopup(teacherSession.LessonDate)
	makeScreenshot("lesson_popup")
	if !assert.NoError(t, err, "Failed to wait for lesson popup") {
		return
	}

	editScoreSelector := `//*[contains(@class, "modal-content")]//a[contains(text(), "оцінок")][contains(text(), "студент")]`

	ctx, cancel := context.WithTimeout(chromeCtx, time.Second*10)
	defer cancel()

	err = chromedp.Run(ctx, chromedp.Tasks{
		chromedp.Click(editScoreSelector),
		chromedp.WaitVisible(`//body`),
	})

	makeScreenshot("edit_score_form")
	verifyLessonOrScoreForm(t)

	scoreInputSelector := `(//input[starts-with(@name, 'st')])[2]`

	inputTypeCtx, inputTypeCancel := context.WithTimeout(ctx, time.Millisecond*500)
	err = chromedp.Run(inputTypeCtx, chromedp.SetValue(scoreInputSelector, "5.5"))
	inputTypeCancel()

	if t.Failed() {
		return
	}

	dekanatReverseProxy.ClearBlockedRequests()
	dekanatReverseProxy.SwitchOffline()
	defer dekanatReverseProxy.SwitchOnline()

	err = chromedp.Run(ctx, chromedp.Tasks{
		chromedp.Click(`//button[contains(text(), "Зберегти")][1]`),
		chromedp.WaitVisible(`//body`),
	})
	assert.NoError(t, err, "Failed to click on 'Зберегти' button")

	// assert
	expectBlockedPage(t)
	assert.Equal(t, 1, len(dekanatReverseProxy.BlockedRequests), "Wrong number of blocked requests")

	event := realtimeQueue.Fetch(time.Second * 15)

	assert.NotNil(t, event, "Event not found")
	assert.IsType(t, dekanatEvents.ScoreEditEvent{}, event, "Wrong event type")

	scoreEditEvent, ok := event.(dekanatEvents.ScoreEditEvent)
	if !ok {
		return
	}

	assert.Equal(t, teacherSession.IsCustomGroup, scoreEditEvent.IsCustomGroup())
	assert.True(t, scoreEditEvent.HasChanges)

	assert.Equal(t, teacherSession.DisciplineId, scoreEditEvent.GetDisciplineId(), "Wrong group id")
	assert.Equal(t, teacherSession.Semester, scoreEditEvent.GetSemester(), "Wrong semester")
	assert.Equal(t, teacherSession.LessonId, scoreEditEvent.GetLessonId(), "Wrong semester")

	expectedLessonDate := teacherSession.LessonDate.Format("02.01.2006")
	assert.Equal(t, expectedLessonDate, scoreEditEvent.Date, "Wrong date")

	realtimeQueue.AssertNoOtherEvents(t)

}
