package main

import (
	"database/sql"
	"fmt"
	_ "github.com/nakagami/firebirdsql"
	"strings"
	"time"
)

type DekanatRepository struct {
	db     *sql.DB
	secret string
}

type TeacherWithActiveLesson struct {
	LessonId       uint
	LessonDate     time.Time
	DisciplineId   uint
	GrpId          uint
	GroupName      string
	DisciplineName string
	Semester       uint
	TeacherId      uint
	Ln             string
	Fn             string
	Pn             string
	Login          string
	Password       string
}

func NewDekanatRepository(dsn string, secret string) (*DekanatRepository, error) {
	db, err := sql.Open("firebirdsql", dsn)
	if err != nil {
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		return nil, err
	}

	return &DekanatRepository{
		db:     db,
		secret: secret,
	}, nil
}

func (r *DekanatRepository) Close() error {
	return r.db.Close()
}

func (r *DekanatRepository) GetTeacherWithActiveLesson() *TeacherWithActiveLesson {
	row := r.db.QueryRow(`
		SELECT FIRST 1
			T_PRJURN.ID,
			T_PRJURN.DATEZAN,
			T_PD_CMS.ID,
			T_PD_CMS.GRP_ID,
			rtrim(TSTGRP.FNAME) AS NAME,
			TPR_COLL.PREDMET,
    		T_PRJURN.HALF,
			T_PRJURN.ID_TEACH,
			TEACHERS_COLL.LN,
			TEACHERS_COLL.FN,
			TEACHERS_COLL.PN,
			(SELECT RES FROM DESHIFR(TSEC.STR1, ?) ) AS "Login",
			(SELECT RES FROM DESHIFR(TSEC.STR2, ?) )AS "Password"
		FROM T_PRJURN
		INNER JOIN T_PD_CMS ON T_PD_CMS.ID = T_PRJURN.NUM_PREDM
		INNER JOIN TPR_COLL ON T_PD_CMS.PREDM_ID = TPR_COLL.ID
		INNER JOIN TSTGRP ON T_PD_CMS.GRP_ID = TSTGRP.ID
		INNER JOIN TEACHERS_COLL ON T_PRJURN.ID_TEACH = TEACHERS_COLL.ID
		INNER JOIN TSEC ON T_PRJURN.ID_TEACH = TSEC.TID
		WHERE T_PRJURN.BLOCKED = 0
		  AND T_PRJURN.FSTATUS = 1
		  AND T_PRJURN.REGDATE > DATEADD(-7 day to CAST((select 'Now' from rdb$database) as date))
		  AND (SELECT FIRST 1 1 FROM T_EV_6 WHERE T_PD_CMS.ID = T_EV_6.XI_2) IS NULL -- семестрові (екземенаційні) бали відсутні
		  AND (SELECT FIRST 1 1 FROM T_CG_LINK WHERE T_CG_LINK.ID_PREDM = T_PD_CMS.ID) IS NULL -- не віртуальна група (не вибіркова дисципліна)
		ORDER BY T_PRJURN.DATEZAN DESC, T_PRJURN.ID DESC
`, r.secret, r.secret)

	if row.Err() != nil {
		fmt.Println("failed to fetch teacher with active lesson", row.Err())
		return nil
	}

	record := &TeacherWithActiveLesson{}

	err := row.Scan(
		&record.LessonId,
		&record.LessonDate,
		&record.DisciplineId,
		&record.GrpId,
		&record.GroupName,
		&record.DisciplineName,
		&record.Semester,
		&record.TeacherId,
		&record.Ln,
		&record.Fn,
		&record.Pn,
		&record.Login,
		&record.Password,
	)

	if err != nil {
		fmt.Println("failed to scan teacher with active lesson", err)
		return nil
	}

	record.GroupName = strings.Trim(record.GroupName, " ")
	record.DisciplineName = strings.Trim(record.DisciplineName, " ")
	record.Ln = strings.Trim(record.Ln, " ")
	record.Fn = strings.Trim(record.Fn, " ")
	record.Pn = strings.Trim(record.Pn, " ")

	return record
}
