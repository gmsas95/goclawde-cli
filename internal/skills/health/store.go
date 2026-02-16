package health

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// Store handles health data persistence
type Store struct {
	db *gorm.DB
}

// NewStore creates a new health store
func NewStore(db *gorm.DB) (*Store, error) {
	store := &Store{db: db}
	
	if err := db.AutoMigrate(&Medication{}, &MedicationLog{}, &HealthMetric{}, &HealthAppointment{}, &HealthGoal{}, &HealthInsight{}); err != nil {
		return nil, fmt.Errorf("failed to migrate health schemas: %w", err)
	}
	
	return store, nil
}

func generateID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return "hlth_" + hex.EncodeToString(bytes)
}

// Medication operations

func (s *Store) CreateMedication(med *Medication) error {
	if med.ID == "" {
		med.ID = generateID()
	}
	
	// Serialize arrays
	if len(med.Times) > 0 {
		timesJSON, _ := json.Marshal(med.Times)
		med.TimesJSON = string(timesJSON)
	}
	if len(med.DaysOfWeek) > 0 {
		daysJSON, _ := json.Marshal(med.DaysOfWeek)
		med.DaysJSON = string(daysJSON)
	}
	
	med.CreatedAt = time.Now()
	med.UpdatedAt = time.Now()
	return s.db.Create(med).Error
}

func (s *Store) GetMedication(id string) (*Medication, error) {
	var med Medication
	err := s.db.Where("id = ?", id).First(&med).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	
	// Deserialize arrays
	if med.TimesJSON != "" {
		json.Unmarshal([]byte(med.TimesJSON), &med.Times)
	}
	if med.DaysJSON != "" {
		json.Unmarshal([]byte(med.DaysJSON), &med.DaysOfWeek)
	}
	
	return &med, err
}

func (s *Store) UpdateMedication(med *Medication) error {
	// Serialize arrays
	if len(med.Times) > 0 {
		timesJSON, _ := json.Marshal(med.Times)
		med.TimesJSON = string(timesJSON)
	}
	if len(med.DaysOfWeek) > 0 {
		daysJSON, _ := json.Marshal(med.DaysOfWeek)
		med.DaysJSON = string(daysJSON)
	}
	
	med.UpdatedAt = time.Now()
	return s.db.Save(med).Error
}

func (s *Store) DeleteMedication(id string) error {
	return s.db.Where("id = ?", id).Delete(&Medication{}).Error
}

func (s *Store) ListMedications(userID string, activeOnly bool) ([]Medication, error) {
	query := s.db.Where("user_id = ?", userID)
	if activeOnly {
		query = query.Where("enabled = ?", true)
	}
	
	var meds []Medication
	err := query.Order("created_at DESC").Find(&meds).Error
	
	// Deserialize arrays
	for i := range meds {
		if meds[i].TimesJSON != "" {
			json.Unmarshal([]byte(meds[i].TimesJSON), &meds[i].Times)
		}
		if meds[i].DaysJSON != "" {
			json.Unmarshal([]byte(meds[i].DaysJSON), &meds[i].DaysOfWeek)
		}
	}
	
	return meds, err
}

// MedicationLog operations

func (s *Store) CreateMedicationLog(log *MedicationLog) error {
	if log.ID == "" {
		log.ID = generateID()
	}
	log.CreatedAt = time.Now()
	return s.db.Create(log).Error
}

func (s *Store) GetMedicationLog(id string) (*MedicationLog, error) {
	var log MedicationLog
	err := s.db.Where("id = ?", id).First(&log).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &log, err
}

func (s *Store) UpdateMedicationLog(log *MedicationLog) error {
	return s.db.Save(log).Error
}

func (s *Store) GetMedicationLogs(userID, medicationID string, start, end time.Time) ([]MedicationLog, error) {
	query := s.db.Where("user_id = ?", userID)
	
	if medicationID != "" {
		query = query.Where("medication_id = ?", medicationID)
	}
	if !start.IsZero() {
		query = query.Where("scheduled_time >= ?", start)
	}
	if !end.IsZero() {
		query = query.Where("scheduled_time <= ?", end)
	}
	
	var logs []MedicationLog
	err := query.Order("scheduled_time DESC").Find(&logs).Error
	return logs, err
}

func (s *Store) GetTodayLogs(userID string) ([]MedicationLog, error) {
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)
	
	return s.GetMedicationLogs(userID, "", startOfDay, endOfDay)
}

// HealthMetric operations

func (s *Store) CreateMetric(metric *HealthMetric) error {
	if metric.ID == "" {
		metric.ID = generateID()
	}
	metric.CreatedAt = time.Now()
	metric.UpdatedAt = time.Now()
	return s.db.Create(metric).Error
}

func (s *Store) GetMetric(id string) (*HealthMetric, error) {
	var metric HealthMetric
	err := s.db.Where("id = ?", id).First(&metric).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &metric, err
}

func (s *Store) UpdateMetric(metric *HealthMetric) error {
	metric.UpdatedAt = time.Now()
	return s.db.Save(metric).Error
}

func (s *Store) DeleteMetric(id string) error {
	return s.db.Where("id = ?", id).Delete(&HealthMetric{}).Error
}

func (s *Store) GetMetrics(userID, metricType string, start, end time.Time, limit int) ([]HealthMetric, error) {
	query := s.db.Where("user_id = ?", userID)
	
	if metricType != "" {
		query = query.Where("type = ?", metricType)
	}
	if !start.IsZero() {
		query = query.Where("measured_at >= ?", start)
	}
	if !end.IsZero() {
		query = query.Where("measured_at <= ?", end)
	}
	if limit > 0 {
		query = query.Limit(limit)
	}
	
	var metrics []HealthMetric
	err := query.Order("measured_at DESC").Find(&metrics).Error
	return metrics, err
}

func (s *Store) GetLatestMetric(userID, metricType string) (*HealthMetric, error) {
	var metric HealthMetric
	err := s.db.Where("user_id = ? AND type = ?", userID, metricType).
		Order("measured_at DESC").
		First(&metric).Error
	
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &metric, err
}

// HealthAppointment operations

func (s *Store) CreateAppointment(appt *HealthAppointment) error {
	if appt.ID == "" {
		appt.ID = generateID()
	}
	appt.CreatedAt = time.Now()
	appt.UpdatedAt = time.Now()
	return s.db.Create(appt).Error
}

func (s *Store) GetAppointment(id string) (*HealthAppointment, error) {
	var appt HealthAppointment
	err := s.db.Where("id = ?", id).First(&appt).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &appt, err
}

func (s *Store) UpdateAppointment(appt *HealthAppointment) error {
	appt.UpdatedAt = time.Now()
	return s.db.Save(appt).Error
}

func (s *Store) DeleteAppointment(id string) error {
	return s.db.Where("id = ?", id).Delete(&HealthAppointment{}).Error
}

func (s *Store) GetUpcomingAppointments(userID string, limit int) ([]HealthAppointment, error) {
	now := time.Now()
	
	query := s.db.Where("user_id = ? AND date_time >= ? AND status IN ?", 
		userID, now, []string{"scheduled", "confirmed"})
	
	if limit > 0 {
		query = query.Limit(limit)
	}
	
	var appts []HealthAppointment
	err := query.Order("date_time ASC").Find(&appts).Error
	return appts, err
}

func (s *Store) GetAppointments(userID string, start, end time.Time) ([]HealthAppointment, error) {
	query := s.db.Where("user_id = ?", userID)
	
	if !start.IsZero() {
		query = query.Where("date_time >= ?", start)
	}
	if !end.IsZero() {
		query = query.Where("date_time <= ?", end)
	}
	
	var appts []HealthAppointment
	err := query.Order("date_time DESC").Find(&appts).Error
	return appts, err
}

// HealthGoal operations

func (s *Store) CreateGoal(goal *HealthGoal) error {
	if goal.ID == "" {
		goal.ID = generateID()
	}
	goal.CreatedAt = time.Now()
	goal.UpdatedAt = time.Now()
	return s.db.Create(goal).Error
}

func (s *Store) GetGoal(id string) (*HealthGoal, error) {
	var goal HealthGoal
	err := s.db.Where("id = ?", id).First(&goal).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &goal, err
}

func (s *Store) UpdateGoal(goal *HealthGoal) error {
	goal.UpdatedAt = time.Now()
	return s.db.Save(goal).Error
}

func (s *Store) DeleteGoal(id string) error {
	return s.db.Where("id = ?", id).Delete(&HealthGoal{}).Error
}

func (s *Store) ListGoals(userID string, status string) ([]HealthGoal, error) {
	query := s.db.Where("user_id = ?", userID)
	
	if status != "" && status != "all" {
		query = query.Where("status = ?", status)
	}
	
	var goals []HealthGoal
	err := query.Order("created_at DESC").Find(&goals).Error
	return goals, err
}

// HealthInsight operations

func (s *Store) CreateInsight(insight *HealthInsight) error {
	if insight.ID == "" {
		insight.ID = generateID()
	}
	insight.CreatedAt = time.Now()
	return s.db.Create(insight).Error
}

func (s *Store) GetInsights(userID string, dismissed bool, limit int) ([]HealthInsight, error) {
	query := s.db.Where("user_id = ? AND dismissed = ?", userID, dismissed)
	
	if limit > 0 {
		query = query.Limit(limit)
	}
	
	var insights []HealthInsight
	err := query.Order("created_at DESC").Find(&insights).Error
	return insights, err
}

func (s *Store) DismissInsight(id string) error {
	now := time.Now()
	return s.db.Model(&HealthInsight{}).Where("id = ?", id).Updates(map[string]interface{}{
		"dismissed":    true,
		"dismissed_at": &now,
	}).Error
}

// Statistics

func (s *Store) GetStats(userID string) (*HealthStats, error) {
	stats := &HealthStats{
		MetricsTracked: []string{},
		LatestMetrics:  make(map[string]interface{}),
	}
	
	var count int64
	
	// Medications
	s.db.Model(&Medication{}).Where("user_id = ?", userID).Count(&count)
	stats.TotalMedications = int(count)
	s.db.Model(&Medication{}).Where("user_id = ? AND enabled = ?", userID, true).Count(&count)
	stats.ActiveMedications = int(count)
	
	// Today's logs
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	
	s.db.Model(&MedicationLog{}).Where("user_id = ? AND scheduled_time >= ? AND status = ?", 
		userID, startOfDay, "taken").Count(&count)
	stats.DosesTakenToday = int(count)
	
	s.db.Model(&MedicationLog{}).Where("user_id = ? AND scheduled_time >= ? AND status = ?", 
		userID, startOfDay, "missed").Count(&count)
	stats.DosesMissedToday = int(count)
	
	// Upcoming appointments
	s.db.Model(&HealthAppointment{}).Where("user_id = ? AND date_time >= ? AND status IN ?",
		userID, now, []string{"scheduled", "confirmed"}).Count(&count)
	stats.UpcomingAppointments = int(count)
	
	// Goals
	s.db.Model(&HealthGoal{}).Where("user_id = ? AND status = ?", userID, "active").Count(&count)
	stats.ActiveGoals = int(count)
	s.db.Model(&HealthGoal{}).Where("user_id = ? AND status = ?", userID, "completed").Count(&count)
	stats.CompletedGoals = int(count)
	
	// Tracked metrics
	var metricTypes []string
	s.db.Model(&HealthMetric{}).Where("user_id = ?", userID).Distinct().Pluck("type", &metricTypes)
	stats.MetricsTracked = metricTypes
	
	// Latest metrics
	for _, mType := range metricTypes {
		metric, _ := s.GetLatestMetric(userID, mType)
		if metric != nil {
			stats.LatestMetrics[mType] = map[string]interface{}{
				"value": metric.Value,
				"unit":  metric.Unit,
				"date":  metric.MeasuredAt,
			}
		}
	}
	
	return stats, nil
}
