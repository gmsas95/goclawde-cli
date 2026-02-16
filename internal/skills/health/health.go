package health

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gmsas95/goclawde-cli/internal/skills"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// HealthSkill provides health tracking capabilities
type HealthSkill struct {
	*skills.BaseSkill
	store  *Store
	parser *Parser
	logger *zap.Logger
}

// NewHealthSkill creates a new health skill
func NewHealthSkill(db *gorm.DB, logger *zap.Logger) (*HealthSkill, error) {
	store, err := NewStore(db)
	if err != nil {
		return nil, fmt.Errorf("failed to create health store: %w", err)
	}

	skill := &HealthSkill{
		BaseSkill: skills.NewBaseSkill("health", "Health Tracking", "1.0.0"),
		store:     store,
		parser:    NewParser(),
		logger:    logger,
	}

	skill.registerTools()
	return skill, nil
}

func (h *HealthSkill) registerTools() {
	tools := []skills.Tool{
		{
			Name:        "add_medication",
			Description: "Add a new medication with schedule. Examples: 'Lisinopril 10mg daily at 8am', 'Metformin 500mg twice daily with meals'",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{
						"type":        "string",
						"description": "Medication name and dosage (e.g., 'Lisinopril 10mg')",
					},
					"schedule": map[string]interface{}{
						"type":        "string",
						"description": "When to take it (e.g., 'daily at 8am', 'twice daily with meals', 'every Monday at 9am')",
					},
					"with_food": map[string]interface{}{
						"type":        "boolean",
						"description": "Whether to take with food",
					},
					"notes": map[string]interface{}{
						"type":        "string",
						"description": "Additional instructions or notes",
					},
				},
				"required": []string{"name", "schedule"},
			},
		},
		{
			Name:        "log_medication",
			Description: "Log that a medication was taken or missed",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"medication_id": map[string]interface{}{
						"type":        "string",
						"description": "ID of the medication",
					},
					"status": map[string]interface{}{
						"type":        "string",
						"enum":        []string{"taken", "missed", "skipped"},
						"description": "Whether it was taken, missed, or skipped",
					},
					"time": map[string]interface{}{
						"type":        "string",
						"description": "When it was taken (default: now)",
					},
					"notes": map[string]interface{}{
						"type":        "string",
						"description": "Any notes about this dose",
					},
				},
				"required": []string{"medication_id", "status"},
			},
		},
		{
			Name:        "list_medications",
			Description: "List all medications",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"active_only": map[string]interface{}{
						"type":        "boolean",
						"description": "Only show active medications",
					},
				},
			},
		},
		{
			Name:        "get_medication_schedule",
			Description: "Get today's medication schedule with what's taken and what's remaining",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"date": map[string]interface{}{
						"type":        "string",
						"description": "Date to check (default: today, format: YYYY-MM-DD)",
					},
				},
			},
		},
		{
			Name:        "add_health_metric",
			Description: "Record a health measurement. Examples: 'Weight 175 lbs', 'Blood pressure 120/80', 'Sleep 7.5 hours', 'Steps 8500'",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"measurement": map[string]interface{}{
						"type":        "string",
						"description": "What was measured (e.g., 'Weight 175 lbs', 'Blood pressure 120/80')",
					},
					"context": map[string]interface{}{
						"type":        "string",
						"description": "Context (morning, evening, before_meal, after_meal, resting)",
					},
					"notes": map[string]interface{}{
						"type":        "string",
						"description": "Additional notes",
					},
				},
				"required": []string{"measurement"},
			},
		},
		{
			Name:        "get_health_metrics",
			Description: "Get health metrics history",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"metric_type": map[string]interface{}{
						"type":        "string",
						"enum":        []string{"weight", "blood_pressure", "heart_rate", "temperature", "blood_sugar", "sleep", "steps", "water", "all"},
						"description": "Type of metric to retrieve",
					},
					"period": map[string]interface{}{
						"type":        "string",
						"enum":        []string{"today", "week", "month", "last_month", "all"},
						"description": "Time period",
					},
				},
			},
		},
		{
			Name:        "add_appointment",
			Description: "Schedule a medical appointment. Examples: 'Doctor checkup tomorrow at 2pm', 'Dentist next Monday at 10am for 1 hour'",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"description": map[string]interface{}{
						"type":        "string",
						"description": "Appointment description (e.g., 'Doctor checkup tomorrow at 2pm')",
					},
					"provider": map[string]interface{}{
						"type":        "string",
						"description": "Doctor or provider name",
					},
					"location": map[string]interface{}{
						"type":        "string",
						"description": "Location or address",
					},
					"notes": map[string]interface{}{
						"type":        "string",
						"description": "Additional notes",
					},
				},
				"required": []string{"description"},
			},
		},
		{
			Name:        "list_appointments",
			Description: "List upcoming medical appointments",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"limit": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum appointments to show",
					},
				},
			},
		},
		{
			Name:        "add_health_goal",
			Description: "Set a health goal. Examples: 'Lose 10 pounds by June', 'Walk 10000 steps daily', 'Sleep 8 hours every night'",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"goal": map[string]interface{}{
						"type":        "string",
						"description": "Goal description (e.g., 'Lose 10 pounds by June 1st')",
					},
					"target_value": map[string]interface{}{
						"type":        "number",
						"description": "Target number (if applicable)",
					},
					"unit": map[string]interface{}{
						"type":        "string",
						"description": "Unit of measurement",
					},
					"target_date": map[string]interface{}{
						"type":        "string",
						"description": "Target date (YYYY-MM-DD)",
					},
				},
				"required": []string{"goal"},
			},
		},
		{
			Name:        "get_health_summary",
			Description: "Get a comprehensive health summary including medications, upcoming appointments, and recent metrics",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"include_medications": map[string]interface{}{
						"type":        "boolean",
						"description": "Include medication summary",
					},
					"include_appointments": map[string]interface{}{
						"type":        "boolean",
						"description": "Include upcoming appointments",
					},
					"include_metrics": map[string]interface{}{
						"type":        "boolean",
						"description": "Include recent metrics",
					},
				},
			},
		},
		{
			Name:        "delete_medication",
			Description: "Remove a medication",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"medication_id": map[string]interface{}{
						"type":        "string",
						"description": "ID of medication to delete",
					},
				},
				"required": []string{"medication_id"},
			},
		},
	}

	for _, tool := range tools {
		tool.Handler = h.handleTool(tool.Name)
		h.AddTool(tool)
	}
}

func (h *HealthSkill) handleTool(name string) skills.ToolHandler {
	return func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		switch name {
		case "add_medication":
			return h.handleAddMedication(ctx, args)
		case "log_medication":
			return h.handleLogMedication(ctx, args)
		case "list_medications":
			return h.handleListMedications(ctx, args)
		case "get_medication_schedule":
			return h.handleGetMedicationSchedule(ctx, args)
		case "add_health_metric":
			return h.handleAddMetric(ctx, args)
		case "get_health_metrics":
			return h.handleGetMetrics(ctx, args)
		case "add_appointment":
			return h.handleAddAppointment(ctx, args)
		case "list_appointments":
			return h.handleListAppointments(ctx, args)
		case "add_health_goal":
			return h.handleAddGoal(ctx, args)
		case "get_health_summary":
			return h.handleGetSummary(ctx, args)
		case "delete_medication":
			return h.handleDeleteMedication(ctx, args)
		default:
			return nil, fmt.Errorf("unknown tool: %s", name)
		}
	}
}

func (h *HealthSkill) getUserID(ctx context.Context) string {
	if userID, ok := ctx.Value("user_id").(string); ok {
		return userID
	}
	return "default_user"
}

func getStringArg(args map[string]interface{}, key string, defaultVal string) string {
	if v, ok := args[key].(string); ok {
		return v
	}
	return defaultVal
}

func getBoolArg(args map[string]interface{}, key string, defaultVal bool) bool {
	if v, ok := args[key].(bool); ok {
		return v
	}
	return defaultVal
}

func (h *HealthSkill) handleAddMedication(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	userID := h.getUserID(ctx)
	
	name := getStringArg(args, "name", "")
	schedule := getStringArg(args, "schedule", "")
	withFood := getBoolArg(args, "with_food", false)
	notes := getStringArg(args, "notes", "")
	
	if name == "" {
		return nil, fmt.Errorf("medication name is required")
	}
	
	// Parse the combined input
	input := name
	if schedule != "" {
		input = name + " " + schedule
	}
	
	parsed := h.parser.ParseMedication(input)
	
	med := &Medication{
		UserID:      userID,
		Name:        parsed.Name,
		Dosage:      parsed.Dosage,
		Form:        parsed.Form,
		Frequency:   parsed.Frequency,
		Times:       parsed.Times,
		DaysOfWeek:  parsed.DaysOfWeek,
		WithFood:    withFood || parsed.WithFood,
		BeforeBed:   parsed.BeforeBed,
		Notes:       notes,
		Enabled:     true,
	}
	
	if err := h.store.CreateMedication(med); err != nil {
		return nil, fmt.Errorf("failed to create medication: %w", err)
	}
	
	h.logger.Info("Medication added",
		zap.String("medication_id", med.ID),
		zap.String("name", med.Name),
	)
	
	return map[string]interface{}{
		"id":        med.ID,
		"name":      med.Name,
		"dosage":    med.Dosage,
		"frequency": med.Frequency,
		"times":     med.Times,
		"message":   fmt.Sprintf("Added %s to your medications", med.Name),
	}, nil
}

func (h *HealthSkill) handleLogMedication(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	userID := h.getUserID(ctx)
	
	medicationID := getStringArg(args, "medication_id", "")
	status := getStringArg(args, "status", "")
	timeStr := getStringArg(args, "time", "")
	notes := getStringArg(args, "notes", "")
	
	if medicationID == "" || status == "" {
		return nil, fmt.Errorf("medication_id and status are required")
	}
	
	// Get medication
	med, err := h.store.GetMedication(medicationID)
	if err != nil {
		return nil, err
	}
	if med == nil {
		return nil, fmt.Errorf("medication not found")
	}
	if med.UserID != userID {
		return nil, fmt.Errorf("unauthorized")
	}
	
	// Parse time
	takenTime := time.Now()
	if timeStr != "" {
		// Try to parse time
		if t, err := time.Parse("15:04", timeStr); err == nil {
			now := time.Now()
			takenTime = time.Date(now.Year(), now.Month(), now.Day(), t.Hour(), t.Minute(), 0, 0, now.Location())
		}
	}
	
	log := &MedicationLog{
		UserID:       userID,
		MedicationID: medicationID,
		Status:       status,
		TakenTime:    &takenTime,
		Notes:        notes,
	}
	
	if err := h.store.CreateMedicationLog(log); err != nil {
		return nil, err
	}
	
	return map[string]interface{}{
		"success":      true,
		"medication":   med.Name,
		"status":       status,
		"time":         takenTime.Format("3:04 PM"),
		"message":      fmt.Sprintf("Logged %s as %s", med.Name, status),
	}, nil
}

func (h *HealthSkill) handleListMedications(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	userID := h.getUserID(ctx)
	activeOnly := getBoolArg(args, "active_only", true)
	
	meds, err := h.store.ListMedications(userID, activeOnly)
	if err != nil {
		return nil, err
	}
	
	var result []map[string]interface{}
	for _, med := range meds {
		schedule := med.Frequency
		if len(med.Times) > 0 {
			schedule += " at " + strings.Join(med.Times, ", ")
		}
		
		result = append(result, map[string]interface{}{
			"id":        med.ID,
			"name":      med.Name,
			"dosage":    med.Dosage,
			"schedule":  schedule,
			"with_food": med.WithFood,
			"enabled":   med.Enabled,
		})
	}
	
	return map[string]interface{}{
		"count":       len(result),
		"medications": result,
	}, nil
}

func (h *HealthSkill) handleGetMedicationSchedule(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	userID := h.getUserID(ctx)
	dateStr := getStringArg(args, "date", "")
	
	// Get date
	checkDate := time.Now()
	if dateStr != "" {
		if d, err := time.Parse("2006-01-02", dateStr); err == nil {
			checkDate = d
		}
	}
	
	// Get medications
	meds, err := h.store.ListMedications(userID, true)
	if err != nil {
		return nil, err
	}
	
	// Get today's logs
	logs, err := h.store.GetTodayLogs(userID)
	if err != nil {
		return nil, err
	}
	
	// Build schedule
	taken := 0
	missed := 0
	remaining := 0
	var schedule []map[string]interface{}
	
	for _, med := range meds {
		for _, t := range med.Times {
			// Parse time
			parts := strings.Split(t, ":")
			hour, _ := strconv.Atoi(parts[0])
			minute, _ := strconv.Atoi(parts[1])
			
			scheduledTime := time.Date(checkDate.Year(), checkDate.Month(), checkDate.Day(), hour, minute, 0, 0, checkDate.Location())
			
			// Check if already logged
			status := "pending"
			for _, log := range logs {
				if log.MedicationID == med.ID {
					// Check if log time is close to scheduled time (within 1 hour)
					diff := log.ScheduledTime.Sub(scheduledTime)
					if diff < time.Hour && diff > -time.Hour {
						status = log.Status
						break
					}
				}
			}
			
			if status == "taken" {
				taken++
			} else if status == "missed" {
				missed++
			} else {
				remaining++
			}
			
			schedule = append(schedule, map[string]interface{}{
				"medication_id":  med.ID,
				"name":           med.Name,
				"dosage":         med.Dosage,
				"scheduled_time": scheduledTime.Format("3:04 PM"),
				"status":         status,
				"with_food":      med.WithFood,
			})
		}
	}
	
	return map[string]interface{}{
		"date":            checkDate.Format("Monday, Jan 2"),
		"total_doses":     len(schedule),
		"taken":           taken,
		"missed":          missed,
		"remaining":       remaining,
		"schedule":        schedule,
	}, nil
}

func (h *HealthSkill) handleAddMetric(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	userID := h.getUserID(ctx)
	
	measurement := getStringArg(args, "measurement", "")
	contextStr := getStringArg(args, "context", "")
	notes := getStringArg(args, "notes", "")
	
	if measurement == "" {
		return nil, fmt.Errorf("measurement is required")
	}
	
	parsed := h.parser.ParseMetric(measurement)
	
	if parsed.Type == "" {
		return nil, fmt.Errorf("could not determine metric type from: %s", measurement)
	}
	
	if parsed.Value == 0 && parsed.Unit == "" {
		return nil, fmt.Errorf("could not parse value from: %s", measurement)
	}
	
	metric := &HealthMetric{
		UserID:    userID,
		Type:      parsed.Type,
		SubType:   parsed.SubType,
		Value:     parsed.Value,
		Unit:      parsed.Unit,
		Context:   contextStr,
		Notes:     notes,
		MeasuredAt: time.Now(),
	}
	
	if err := h.store.CreateMetric(metric); err != nil {
		return nil, err
	}
	
	return map[string]interface{}{
		"id":     metric.ID,
		"type":   metric.Type,
		"value":  metric.Value,
		"unit":   metric.Unit,
		"message": fmt.Sprintf("Recorded %s: %.1f %s", metric.Type, metric.Value, metric.Unit),
	}, nil
}

func (h *HealthSkill) handleGetMetrics(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	userID := h.getUserID(ctx)
	
	metricType := getStringArg(args, "metric_type", "all")
	period := getStringArg(args, "period", "week")
	
	// Calculate date range
	now := time.Now()
	var start time.Time
	
	switch period {
	case "today":
		start = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	case "week":
		start = now.AddDate(0, 0, -7)
	case "month":
		start = now.AddDate(0, 0, -30)
	case "last_month":
		start = time.Date(now.Year(), now.Month()-1, 1, 0, 0, 0, 0, now.Location())
		now = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	default:
		start = now.AddDate(0, 0, -7)
	}
	
	metrics, err := h.store.GetMetrics(userID, metricType, start, now, 0)
	if err != nil {
		return nil, err
	}
	
	var result []map[string]interface{}
	for _, m := range metrics {
		result = append(result, map[string]interface{}{
			"id":          m.ID,
			"type":        m.Type,
			"value":       m.Value,
			"unit":        m.Unit,
			"measured_at": m.MeasuredAt.Format("Jan 2, 3:04 PM"),
			"context":     m.Context,
		})
	}
	
	return map[string]interface{}{
		"period":  period,
		"count":   len(result),
		"metrics": result,
	}, nil
}

func (h *HealthSkill) handleAddAppointment(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	userID := h.getUserID(ctx)
	
	description := getStringArg(args, "description", "")
	provider := getStringArg(args, "provider", "")
	location := getStringArg(args, "location", "")
	notes := getStringArg(args, "notes", "")
	
	if description == "" {
		return nil, fmt.Errorf("appointment description is required")
	}
	
	parsed := h.parser.ParseAppointment(description)
	
	appt := &HealthAppointment{
		UserID:       userID,
		Title:        parsed.Title,
		Description:  description,
		Type:         parsed.Type,
		ProviderName: provider,
		Specialty:    parsed.Specialty,
		Location:     location,
		DateTime:     parsed.DateTime,
		Duration:     parsed.Duration,
		Status:       "scheduled",
		Notes:        notes,
	}
	
	if appt.ProviderName == "" {
		appt.ProviderName = parsed.Provider
	}
	if appt.Location == "" {
		appt.Location = parsed.Location
	}
	
	if err := h.store.CreateAppointment(appt); err != nil {
		return nil, err
	}
	
	return map[string]interface{}{
		"id":       appt.ID,
		"title":    appt.Title,
		"date":     appt.DateTime.Format("Monday, Jan 2 at 3:04 PM"),
		"provider": appt.ProviderName,
		"location": appt.Location,
		"message":  fmt.Sprintf("Scheduled %s for %s", appt.Title, appt.DateTime.Format("Monday, Jan 2")),
	}, nil
}

func (h *HealthSkill) handleListAppointments(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	userID := h.getUserID(ctx)
	
	limit := 10
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
	}
	
	appts, err := h.store.GetUpcomingAppointments(userID, limit)
	if err != nil {
		return nil, err
	}
	
	var result []map[string]interface{}
	for _, appt := range appts {
		result = append(result, map[string]interface{}{
			"id":       appt.ID,
			"title":    appt.Title,
			"date":     appt.DateTime.Format("Monday, Jan 2 at 3:04 PM"),
			"provider": appt.ProviderName,
			"location": appt.Location,
			"type":     appt.Type,
		})
	}
	
	return map[string]interface{}{
		"count":        len(result),
		"appointments": result,
	}, nil
}

func (h *HealthSkill) handleAddGoal(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	userID := h.getUserID(ctx)
	
	goalText := getStringArg(args, "goal", "")
	targetValue := 0.0
	if v, ok := args["target_value"].(float64); ok {
		targetValue = v
	}
	unit := getStringArg(args, "unit", "")
	targetDateStr := getStringArg(args, "target_date", "")
	
	if goalText == "" {
		return nil, fmt.Errorf("goal description is required")
	}
	
	goal := &HealthGoal{
		UserID:    userID,
		Title:     goalText,
		Type:      "general",
		TargetValue: targetValue,
		Unit:      unit,
		StartDate: time.Now(),
		Status:    "active",
	}
	
	if targetDateStr != "" {
		if d, err := time.Parse("2006-01-02", targetDateStr); err == nil {
			goal.TargetDate = &d
		}
	}
	
	if err := h.store.CreateGoal(goal); err != nil {
		return nil, err
	}
	
	return map[string]interface{}{
		"id":          goal.ID,
		"title":       goal.Title,
		"target_date": targetDateStr,
		"message":     fmt.Sprintf("Created goal: %s", goal.Title),
	}, nil
}

func (h *HealthSkill) handleGetSummary(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	userID := h.getUserID(ctx)
	
	includeMeds := getBoolArg(args, "include_medications", true)
	includeAppts := getBoolArg(args, "include_appointments", true)
	includeMetrics := getBoolArg(args, "include_metrics", true)
	
	result := map[string]interface{}{
		"generated_at": time.Now().Format("Jan 2, 3:04 PM"),
	}
	
	// Medication summary
	if includeMeds {
		meds, _ := h.store.ListMedications(userID, true)
		logs, _ := h.store.GetTodayLogs(userID)
		
		taken := 0
		for _, log := range logs {
			if log.Status == "taken" {
				taken++
			}
		}
		
		result["medications"] = map[string]interface{}{
			"active_count":  len(meds),
			"doses_today":   taken,
			"total_doses":   len(logs),
		}
	}
	
	// Appointments
	if includeAppts {
		appts, _ := h.store.GetUpcomingAppointments(userID, 5)
		
		var apptList []map[string]interface{}
		for _, appt := range appts {
			apptList = append(apptList, map[string]interface{}{
				"title": appt.Title,
				"date":  appt.DateTime.Format("Mon, Jan 2"),
			})
		}
		
		result["upcoming_appointments"] = apptList
	}
	
	// Recent metrics
	if includeMetrics {
		latestMetrics := make(map[string]interface{})
		
		for _, mType := range []string{"weight", "blood_pressure", "heart_rate", "sleep"} {
			metric, _ := h.store.GetLatestMetric(userID, mType)
			if metric != nil {
				latestMetrics[mType] = map[string]interface{}{
					"value": metric.Value,
					"unit":  metric.Unit,
					"date":  metric.MeasuredAt.Format("Jan 2"),
				}
			}
		}
		
		result["latest_metrics"] = latestMetrics
	}
	
	return result, nil
}

func (h *HealthSkill) handleDeleteMedication(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	userID := h.getUserID(ctx)
	
	medicationID := getStringArg(args, "medication_id", "")
	if medicationID == "" {
		return nil, fmt.Errorf("medication_id is required")
	}
	
	// Verify ownership
	med, err := h.store.GetMedication(medicationID)
	if err != nil {
		return nil, err
	}
	if med == nil {
		return nil, fmt.Errorf("medication not found")
	}
	if med.UserID != userID {
		return nil, fmt.Errorf("unauthorized")
	}
	
	if err := h.store.DeleteMedication(medicationID); err != nil {
		return nil, err
	}
	
	return map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Deleted %s", med.Name),
	}, nil
}
