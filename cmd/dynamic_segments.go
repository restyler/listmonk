package main

import (
	"net/http"
	"strconv"

	"github.com/knadh/listmonk/internal/auth"
	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo/v4"
)

// HandleGetDynamicSegments handles the retrieval of dynamic segments.
func (a *App) HandleGetDynamicSegments(c echo.Context) error {
	// Get the authenticated user.
	user := auth.GetUser(c)

	// Check permissions
	if !user.HasPerm(auth.PermSubscribersSqlQuery) {
		return echo.NewHTTPError(http.StatusForbidden,
			a.i18n.Ts("globals.messages.permissionDenied", "name", auth.PermSubscribersSqlQuery))
	}

	var (
		pg           = a.pg.NewFromURL(c.Request().URL.Query())
		id, _        = strconv.Atoi(c.Param("id"))
		uuid         = c.QueryParam("uuid")
		listID, _    = strconv.Atoi(c.QueryParam("list_id"))
		snippetID, _ = strconv.Atoi(c.QueryParam("snippet_id"))
		isActive     *bool
	)

	if v := c.QueryParam("is_active"); v != "" {
		if val, err := strconv.ParseBool(v); err == nil {
			isActive = &val
		}
	}

	// Single segment by ID.
	if id > 0 {
		out, err := a.core.GetDynamicSegment(id, "")
		if err != nil {
			return err
		}
		return c.JSON(http.StatusOK, okResp{out})
	}

	// Multiple segments.
	out, err := a.core.GetDynamicSegments(0, uuid, listID, snippetID, isActive, pg.Offset, pg.Limit)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{models.PageResults{
		Results: out,
		Page:    pg.Page,
		PerPage: pg.PerPage,
		Total:   len(out),
	}})
}

// HandleCreateDynamicSegment handles the creation of a new dynamic segment.
func (a *App) HandleCreateDynamicSegment(c echo.Context) error {
	// Get the authenticated user.
	user := auth.GetUser(c)

	// Check permissions
	if !user.HasPerm(auth.PermSubscribersSqlQuery) {
		return echo.NewHTTPError(http.StatusForbidden,
			a.i18n.Ts("globals.messages.permissionDenied", "name", auth.PermSubscribersSqlQuery))
	}

	var ds models.DynamicSegment
	if err := c.Bind(&ds); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	// Validate that the list exists and user has permission
	listIDs, err := a.filterListQueryByPerm("list_id", map[string][]string{
		"list_id": {strconv.Itoa(ds.ListID)},
	}, user)
	if err != nil || len(listIDs) == 0 || listIDs[0] != ds.ListID {
		return echo.NewHTTPError(http.StatusForbidden, a.i18n.T("globals.messages.permissionDenied"))
	}

	out, err := a.core.CreateDynamicSegment(ds, user.ID)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// HandleUpdateDynamicSegment handles the update of a dynamic segment.
func (a *App) HandleUpdateDynamicSegment(c echo.Context) error {
	// Get the authenticated user.
	user := auth.GetUser(c)

	// Check permissions
	if !user.HasPerm(auth.PermSubscribersSqlQuery) {
		return echo.NewHTTPError(http.StatusForbidden,
			a.i18n.Ts("globals.messages.permissionDenied", "name", auth.PermSubscribersSqlQuery))
	}

	id := getID(c)
	if id < 1 {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("globals.messages.invalidID"))
	}

	var ds models.DynamicSegment
	if err := c.Bind(&ds); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	// If list ID is being updated, validate that the list exists and user has permission
	if ds.ListID > 0 {
		listIDs, err := a.filterListQueryByPerm("list_id", map[string][]string{
			"list_id": {strconv.Itoa(ds.ListID)},
		}, user)
		if err != nil || len(listIDs) == 0 || listIDs[0] != ds.ListID {
			return echo.NewHTTPError(http.StatusForbidden, a.i18n.T("globals.messages.permissionDenied"))
		}
	}

	out, err := a.core.UpdateDynamicSegment(id, ds)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// HandleDeleteDynamicSegment handles the deletion of a dynamic segment.
func (a *App) HandleDeleteDynamicSegment(c echo.Context) error {
	// Get the authenticated user.
	user := auth.GetUser(c)

	// Check permissions
	if !user.HasPerm(auth.PermSubscribersSqlQuery) {
		return echo.NewHTTPError(http.StatusForbidden,
			a.i18n.Ts("globals.messages.permissionDenied", "name", auth.PermSubscribersSqlQuery))
	}

	id := getID(c)
	if id < 1 {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("globals.messages.invalidID"))
	}

	if err := a.core.DeleteDynamicSegment(id); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// HandleExecuteDynamicSegment handles manual execution of a dynamic segment.
func (a *App) HandleExecuteDynamicSegment(c echo.Context) error {
	// Get the authenticated user.
	user := auth.GetUser(c)

	// Check permissions
	if !user.HasPerm(auth.PermSubscribersSqlQuery) {
		return echo.NewHTTPError(http.StatusForbidden,
			a.i18n.Ts("globals.messages.permissionDenied", "name", auth.PermSubscribersSqlQuery))
	}

	id := getID(c)
	if id < 1 {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("globals.messages.invalidID"))
	}

	// Get the segment
	segment, err := a.core.GetDynamicSegment(id, "")
	if err != nil {
		return err
	}

	// Execute the segment
	if err := a.core.ExecuteDynamicSegment(segment); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{map[string]interface{}{
		"message": "Dynamic segment executed successfully",
	}})
}

// HandleExecuteAllDynamicSegments handles manual execution of all active dynamic segments.
func (a *App) HandleExecuteAllDynamicSegments(c echo.Context) error {
	// Get the authenticated user.
	user := auth.GetUser(c)

	// Check permissions
	if !user.HasPerm(auth.PermSubscribersSqlQuery) {
		return echo.NewHTTPError(http.StatusForbidden,
			a.i18n.Ts("globals.messages.permissionDenied", "name", auth.PermSubscribersSqlQuery))
	}

	// Execute all segments
	if err := a.core.ExecuteAllDynamicSegments(); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{map[string]interface{}{
		"message": "All dynamic segments executed successfully",
	}})
}

// HandleGetDynamicSegmentRuns handles the retrieval of dynamic segment runs.
func (a *App) HandleGetDynamicSegmentRuns(c echo.Context) error {
	// Get the authenticated user.
	user := auth.GetUser(c)

	// Check permissions
	if !user.HasPerm(auth.PermSubscribersSqlQuery) {
		return echo.NewHTTPError(http.StatusForbidden,
			a.i18n.Ts("globals.messages.permissionDenied", "name", auth.PermSubscribersSqlQuery))
	}

	var (
		pg           = a.pg.NewFromURL(c.Request().URL.Query())
		segmentID, _ = strconv.Atoi(c.QueryParam("segment_id"))
	)

	out, err := a.core.GetDynamicSegmentRuns(segmentID, pg.Offset, pg.Limit)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{models.PageResults{
		Results: out,
		Page:    pg.Page,
		PerPage: pg.PerPage,
		Total:   len(out),
	}})
}
