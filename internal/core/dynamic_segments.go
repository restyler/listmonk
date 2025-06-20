package core

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo/v4"
	"github.com/lib/pq"
)

// DynamicSegmentStats represents statistics from a dynamic segment run.
type DynamicSegmentStats struct {
	AddedCount      int    `json:"added_count"`
	RemovedCount    int    `json:"removed_count"`
	TotalMatched    int    `json:"total_matched"`
	ExecutionTimeMs int    `json:"execution_time_ms"`
	Status          string `json:"status"`
	ErrorMessage    string `json:"error_message,omitempty"`
}

// GetDynamicSegments retrieves dynamic segments based on the given filters.
func (c *Core) GetDynamicSegments(id int, uuid string, listID, snippetID int, isActive *bool, offset, limit int) ([]models.DynamicSegment, error) {
	var out []models.DynamicSegment
	if err := c.q.GetDynamicSegments.Select(&out, id, uuid, listID, snippetID, isActive, offset, limit); err != nil {
		c.log.Printf("error fetching dynamic segments: %v", err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "dynamic segments", "error", pqErrMsg(err)))
	}

	return out, nil
}

// GetDynamicSegment retrieves a single dynamic segment by ID or UUID.
func (c *Core) GetDynamicSegment(id int, uuid string) (models.DynamicSegment, error) {
	var out models.DynamicSegment
	if err := c.q.GetDynamicSegment.Get(&out, id, uuid); err != nil {
		c.log.Printf("error fetching dynamic segment: %v", err)
		return models.DynamicSegment{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "dynamic segment", "error", pqErrMsg(err)))
	}

	return out, nil
}

// CreateDynamicSegment creates a new dynamic segment.
func (c *Core) CreateDynamicSegment(ds models.DynamicSegment, createdBy int) (models.DynamicSegment, error) {
	uu, err := uuid.NewV4()
	if err != nil {
		c.log.Printf("error generating UUID: %v", err)
		return models.DynamicSegment{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorUUID", "error", err.Error()))
	}

	var createdByVal *int
	if createdBy > 0 {
		createdByVal = &createdBy
	}

	var newID int
	if err := c.q.CreateDynamicSegment.Get(&newID, uu, ds.Name, ds.Description, ds.ListID, ds.SnippetID, createdByVal); err != nil {
		c.log.Printf("error creating dynamic segment: %v", err)
		return models.DynamicSegment{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorCreating", "name", "dynamic segment", "error", pqErrMsg(err)))
	}

	return c.GetDynamicSegment(newID, "")
}

// UpdateDynamicSegment updates an existing dynamic segment.
func (c *Core) UpdateDynamicSegment(id int, ds models.DynamicSegment) (models.DynamicSegment, error) {
	_, err := c.q.UpdateDynamicSegment.Exec(id, ds.Name, ds.Description, ds.ListID, ds.SnippetID, ds.IsActive)
	if err != nil {
		c.log.Printf("error updating dynamic segment: %v", err)
		return models.DynamicSegment{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorUpdating", "name", "dynamic segment", "error", pqErrMsg(err)))
	}

	return c.GetDynamicSegment(id, "")
}

// DeleteDynamicSegment deletes a dynamic segment.
func (c *Core) DeleteDynamicSegment(id int) error {
	_, err := c.q.DeleteDynamicSegment.Exec(id)
	if err != nil {
		c.log.Printf("error deleting dynamic segment: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorDeleting", "name", "dynamic segment", "error", pqErrMsg(err)))
	}

	return nil
}

// GetActiveDynamicSegments retrieves all active dynamic segments.
func (c *Core) GetActiveDynamicSegments() ([]models.DynamicSegment, error) {
	var out []models.DynamicSegment
	if err := c.q.GetActiveDynamicSegments.Select(&out); err != nil {
		c.log.Printf("error fetching active dynamic segments: %v", err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "active dynamic segments", "error", pqErrMsg(err)))
	}

	return out, nil
}

// GetDynamicSegmentRuns retrieves runs for a dynamic segment.
func (c *Core) GetDynamicSegmentRuns(segmentID, offset, limit int) ([]models.DynamicSegmentRun, error) {
	var out []models.DynamicSegmentRun
	if err := c.q.GetDynamicSegmentRuns.Select(&out, segmentID, offset, limit); err != nil {
		c.log.Printf("error fetching dynamic segment runs: %v", err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "dynamic segment runs", "error", pqErrMsg(err)))
	}

	return out, nil
}

// ExecuteDynamicSegment executes a single dynamic segment.
func (c *Core) ExecuteDynamicSegment(segment models.DynamicSegment) error {
	start := time.Now()
	c.log.Printf("executing dynamic segment: %s (ID: %d)", segment.Name, segment.ID)

	stats := DynamicSegmentStats{
		Status: "success",
	}

	// Get current subscribers in the list
	currentSubs, err := c.getListSubscriberIDs(segment.ListID)
	if err != nil {
		return c.recordSegmentRun(segment.ID, DynamicSegmentStats{
			Status:          "error",
			ErrorMessage:    fmt.Sprintf("Failed to get current subscribers: %v", err),
			ExecutionTimeMs: int(time.Since(start).Milliseconds()),
		})
	}

	// Get matching subscribers from the SQL query
	matchingSubs, err := c.getMatchingSubscriberIDs(segment.QuerySQL)
	if err != nil {
		return c.recordSegmentRun(segment.ID, DynamicSegmentStats{
			Status:          "error",
			ErrorMessage:    fmt.Sprintf("Failed to execute query: %v", err),
			ExecutionTimeMs: int(time.Since(start).Milliseconds()),
		})
	}

	stats.TotalMatched = len(matchingSubs)

	// Calculate differences
	toAdd := difference(matchingSubs, currentSubs)
	toRemove := difference(currentSubs, matchingSubs)

	stats.AddedCount = len(toAdd)
	stats.RemovedCount = len(toRemove)

	// Add new subscribers to the list
	if len(toAdd) > 0 {
		if err := c.addSubscribersToList(toAdd, segment.ListID); err != nil {
			return c.recordSegmentRun(segment.ID, DynamicSegmentStats{
				Status:          "error",
				ErrorMessage:    fmt.Sprintf("Failed to add subscribers: %v", err),
				ExecutionTimeMs: int(time.Since(start).Milliseconds()),
			})
		}
	}

	// Remove subscribers from the list
	if len(toRemove) > 0 {
		if err := c.removeSubscribersFromList(toRemove, segment.ListID); err != nil {
			return c.recordSegmentRun(segment.ID, DynamicSegmentStats{
				Status:          "error",
				ErrorMessage:    fmt.Sprintf("Failed to remove subscribers: %v", err),
				ExecutionTimeMs: int(time.Since(start).Milliseconds()),
			})
		}
	}

	stats.ExecutionTimeMs = int(time.Since(start).Milliseconds())

	// Update segment stats
	if err := c.updateSegmentStats(segment.ID, stats); err != nil {
		c.log.Printf("error updating segment stats: %v", err)
	}

	// Record the run
	if err := c.recordSegmentRun(segment.ID, stats); err != nil {
		c.log.Printf("error recording segment run: %v", err)
	}

	c.log.Printf("completed dynamic segment: %s (ID: %d) - added: %d, removed: %d, total matched: %d, time: %dms",
		segment.Name, segment.ID, stats.AddedCount, stats.RemovedCount, stats.TotalMatched, stats.ExecutionTimeMs)

	return nil
}

// ExecuteAllDynamicSegments executes all active dynamic segments.
func (c *Core) ExecuteAllDynamicSegments() error {
	segments, err := c.GetActiveDynamicSegments()
	if err != nil {
		return err
	}

	c.log.Printf("executing %d dynamic segments", len(segments))

	for _, segment := range segments {
		if err := c.ExecuteDynamicSegment(segment); err != nil {
			c.log.Printf("error executing dynamic segment %s: %v", segment.Name, err)
			continue
		}
	}

	c.log.Printf("completed executing all dynamic segments")
	return nil
}

// Helper functions

func (c *Core) getListSubscriberIDs(listID int) ([]int, error) {
	var ids []int
	err := c.db.Select(&ids, `
		SELECT subscriber_id FROM subscriber_lists 
		WHERE list_id = $1 AND status = 'confirmed'
	`, listID)
	return ids, err
}

func (c *Core) getMatchingSubscriberIDs(querySQL string) ([]int, error) {
	// Build the query using the same pattern as QuerySubscribers
	stmt := strings.ReplaceAll(c.q.QuerySubscribersTpl, "%query%", querySQL)

	// Validate the query
	if err := validateQueryTables(c.db, stmt, allowedSubQueryTables); err != nil {
		return nil, err
	}

	tx, err := c.db.BeginTxx(context.Background(), &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var ids []int
	// Execute the query with empty filters to get all matching subscribers
	err = tx.Select(&ids, stmt, pq.Array([]int{}), "", "", "")
	return ids, err
}

func (c *Core) addSubscribersToList(subscriberIDs []int, listID int) error {
	_, err := c.q.AddSubscribersToLists.Exec(pq.Array(subscriberIDs), pq.Array([]int{listID}), "confirmed")
	return err
}

func (c *Core) removeSubscribersFromList(subscriberIDs []int, listID int) error {
	if len(subscriberIDs) == 0 {
		return nil
	}

	query := `
		DELETE FROM subscriber_lists 
		WHERE subscriber_id = ANY($1) AND list_id = $2
	`
	_, err := c.db.Exec(query, pq.Array(subscriberIDs), listID)
	return err
}

func (c *Core) updateSegmentStats(segmentID int, stats DynamicSegmentStats) error {
	statsJSON, err := json.Marshal(stats)
	if err != nil {
		return err
	}

	_, err = c.q.UpdateDynamicSegmentRunStats.Exec(segmentID, statsJSON)
	return err
}

func (c *Core) recordSegmentRun(segmentID int, stats DynamicSegmentStats) error {
	_, err := c.q.CreateDynamicSegmentRun.Exec(
		segmentID,
		stats.AddedCount,
		stats.RemovedCount,
		stats.TotalMatched,
		stats.ExecutionTimeMs,
		stats.Status,
		nullableString(stats.ErrorMessage),
	)
	return err
}

// difference returns elements in a that are not in b
func difference(a, b []int) []int {
	mb := make(map[int]bool, len(b))
	for _, x := range b {
		mb[x] = true
	}

	var diff []int
	for _, x := range a {
		if !mb[x] {
			diff = append(diff, x)
		}
	}
	return diff
}

// nullableString returns nil if s is empty, otherwise returns s
func nullableString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
