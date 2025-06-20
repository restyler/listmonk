package core

import (
	"fmt"
	"net/http"

	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo/v4"
)

// GetSQLSnippets retrieves SQL snippets based on the given filters.
func (c *Core) GetSQLSnippets(id int, name string, isActive *bool, offset, limit int) ([]models.SQLSnippet, error) {
	var out []models.SQLSnippet
	if err := c.q.GetSQLSnippets.Select(&out, id, name, isActive, offset, limit); err != nil {
		c.log.Printf("error fetching SQL snippets: %v", err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "SQL snippets", "error", pqErrMsg(err)))
	}

	return out, nil
}

// GetSQLSnippet retrieves a single SQL snippet by ID or name.
func (c *Core) GetSQLSnippet(id int, name string) (models.SQLSnippet, error) {
	var out models.SQLSnippet
	if err := c.q.GetSQLSnippet.Get(&out, id, name); err != nil {
		c.log.Printf("error fetching SQL snippet: %v", err)
		return models.SQLSnippet{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "SQL snippet", "error", pqErrMsg(err)))
	}

	return out, nil
}

// CreateSQLSnippet creates a new SQL snippet.
func (c *Core) CreateSQLSnippet(s models.SQLSnippet, createdBy int) (models.SQLSnippet, error) {
	var createdByVal *int
	if createdBy > 0 {
		createdByVal = &createdBy
	}

	var newID int
	if err := c.q.CreateSQLSnippet.Get(&newID, s.Name, s.Description, s.QuerySQL, createdByVal); err != nil {
		c.log.Printf("error creating SQL snippet: %v", err)
		return models.SQLSnippet{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorCreating", "name", "SQL snippet", "error", pqErrMsg(err)))
	}

	return c.GetSQLSnippet(newID, "")
}

// UpdateSQLSnippet updates an existing SQL snippet.
func (c *Core) UpdateSQLSnippet(id int, s models.SQLSnippet) (models.SQLSnippet, error) {
	_, err := c.q.UpdateSQLSnippet.Exec(id, s.Name, s.Description, s.QuerySQL, s.IsActive)
	if err != nil {
		c.log.Printf("error updating SQL snippet: %v", err)
		return models.SQLSnippet{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorUpdating", "name", "SQL snippet", "error", pqErrMsg(err)))
	}

	return c.GetSQLSnippet(id, "")
}

// DeleteSQLSnippet deletes an SQL snippet.
func (c *Core) DeleteSQLSnippet(id int) error {
	_, err := c.q.DeleteSQLSnippet.Exec(id)
	if err != nil {
		c.log.Printf("error deleting SQL snippet: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorDeleting", "name", "SQL snippet", "error", pqErrMsg(err)))
	}

	return nil
}

// ValidateSQLSnippet validates an SQL snippet query for safety and syntax.
func (c *Core) ValidateSQLSnippet(querySQL string) error {
	// Create the actual query by replacing the placeholder
	stmt := fmt.Sprintf(`
		SELECT COUNT(*) FROM subscribers
		LEFT JOIN subscriber_lists ON (
			(CASE WHEN CARDINALITY($1::INT[]) > 0 THEN true ELSE false END)
			AND subscriber_lists.subscriber_id = subscribers.id
			AND ($2 = '' OR subscriber_lists.status = $2::subscription_status)
		)
		WHERE (CARDINALITY($1) = 0 OR subscriber_lists.list_id = ANY($1::INT[]))
		AND (CASE WHEN $3 != '' THEN name ~* $3 OR email ~* $3 ELSE TRUE END)
		AND %s
	`, querySQL)

	// Validate the query by running a count operation
	if err := validateQueryTables(c.db, stmt, allowedSubQueryTables); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest,
			c.i18n.Ts("subscribers.errorPreparingQuery", "error", err.Error()))
	}

	return nil
}
