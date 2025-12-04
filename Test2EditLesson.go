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

func Test2EditLesson(t *testing.T) {
	fmt.Println("Test2EditLesson")
	defer printTestResult(t, "Test2EditLesson")

	err := chooseDiscipline()
	if !assert.NoError(t, err, "Failed to choose discipline") {
		return
	}

	err = openLessonPopup(teacherSession.LessonDate)
	makeScreenshot("lesson_popup")
	if !assert.NoError(t, err, "Failed to wait for lesson popup") {
		return
	}

	editLessonSelector := `//*[contains(@class, "modal-content")]//a[contains(text(), "Змінити загальні дані")]`

	ctx, cancel := context.WithTimeout(chromeCtx, time.Second*10)
	defer cancel()

	err = chromedp.Run(ctx, chromedp.Tasks{
		chromedp.Click(editLessonSelector),
		chromedp.WaitVisible(`//body`),
	})

	verifyLessonOrScoreForm(t)
	makeScreenshot("edit_lesson_form")

	radioClickCtx, radioClickCancel := context.WithTimeout(ctx, time.Second*2)
	err = chromedp.Run(radioClickCtx, chromedp.Click(`(//*[@name ="tzn" and not(@checked)])[1]`))
	radioClickCancel()

	assert.NoError(t, err, "Failed to click on `Вид заняття` radio button")

	if t.Failed() {
		fmt.Println("Failed to click on `Вид заняття` radio button")
		return
	}

	fmt.Println("Clicked on `Вид заняття` radio button")
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
	assert.IsType(t, dekanatEvents.LessonEditEvent{}, event, "Wrong event type")

	lessonEditEvent, ok := event.(dekanatEvents.LessonEditEvent)
	if !ok {
		return
	}

	assert.True(t, lessonEditEvent.HasChanges)
	assert.Equal(t, teacherSession.IsCustomGroup, lessonEditEvent.IsCustomGroup())

	assert.Equal(t, teacherSession.DisciplineId, lessonEditEvent.GetDisciplineId(), "Wrong group id")
	assert.Equal(t, teacherSession.Semester, lessonEditEvent.GetSemester(), "Wrong semester")
	assert.Equal(t, teacherSession.LessonId, lessonEditEvent.GetLessonId(), "Wrong lesson id")

	if t.Failed() {
		fmt.Printf("Wrong event: %+v; expected %+v\n", lessonEditEvent, teacherSession)
		return
	}

	expectedLessonDate := teacherSession.LessonDate.Format("02.01.2006")
	assert.Equal(t, expectedLessonDate, lessonEditEvent.Date, "Wrong date")

	realtimeQueue.AssertNoOtherEvents(t)
}
