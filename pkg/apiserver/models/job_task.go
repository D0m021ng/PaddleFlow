package models

import (
	"encoding/json"
	"fmt"
	"paddleflow/pkg/common/schema"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"paddleflow/pkg/common/database"
	"paddleflow/pkg/common/logger"
)

const (
	JobTaskTableName = "job_task"
)

type JobTask struct {
	Pk                   int64             `json:"-" gorm:"primaryKey;autoIncrement"`
	ID                   string            `json:"id" gorm:"type:varchar(64);uniqueIndex"` // k8s:podID
	JobID                string            `json:"jobID" gorm:"type:varchar(60)"`
	Namespace            string            `json:"namespace" gorm:"type:varchar(64)"`
	Name                 string            `json:"name" gorm:"type:varchar(512)"`
	MemberRole           schema.MemberRole `json:"memberRole"`
	Status               schema.TaskStatus `json:"status"`
	Message              string            `json:"message"`
	LogURL               string            `json:"logURL"`
	ExtRuntimeStatusJSON string            `json:"extRuntimeStatus" gorm:"column:ext_runtime_status;default:'{}'"`
	ExtRuntimeStatus     interface{}       `json:"-" gorm:"-"` //k8s:v1.PodStatus
	CreatedAt            time.Time         `json:"-"`
	StartedAt            time.Time         `json:"-"`
	UpdatedAt            time.Time         `json:"-"`
	DeletedAt            time.Time         `json:"-"`
}

func (JobTask) TableName() string {
	return JobTaskTableName
}

func (task *JobTask) BeforeSave(*gorm.DB) error {
	if task.ExtRuntimeStatus != nil {
		statusJSON, err := json.Marshal(task.ExtRuntimeStatus)
		if err != nil {
			return err
		}
		task.ExtRuntimeStatusJSON = string(statusJSON)
	}
	return nil
}

func (task *JobTask) AfterFind(*gorm.DB) error {
	return nil
}

func GetJobTaskByID(id string) (JobTask, error) {
	var taskStatus JobTask
	tx := database.DB.Table(JobTaskTableName).Where("id = ?", id).First(&taskStatus)
	if tx.Error != nil {
		logger.LoggerForJob(id).Errorf("get job task status failed, err %v", tx.Error.Error())
		return JobTask{}, tx.Error
	}
	return taskStatus, nil
}

func UpdateTask(task *JobTask) error {
	if task == nil {
		return fmt.Errorf("JobTask is nil")
	}
	tx := database.DB.Table(JobTaskTableName).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"status", "message", "ext_runtime_status", "deleted_at"}),
	}).Create(task)
	return tx.Error
}