package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	jsonpatch "github.com/evanphx/json-patch"
	"github.com/labstack/echo/v4"
)

type User struct {
	ID      int64  `json:"id,omitempty"`
	Name    string `json:"name,omitempty"`
	Age     int    `json:"age,omitempty"`
	IsAdult bool   `json:"is_adult,omitempty"`
}

type Event struct {
	ID        int64     `json:"id,omitempty"`
	CreatedAt time.Time `json:"created_at,omitempty"`
	Initiator string    `json:"initiator,omitempty"`
	Subject   string    `json:"subject,omitempty"`
	Action    string    `json:"action,omitempty"`
	Rollback  any       `json:"rollback,omitempty"`
	Update    any       `json:"update,omitempty"`
}

const (
	rollback = "rollback"
	update   = "update"
)

var (
	users = map[int64]*User{
		1: {ID: 1, Name: "John", Age: 16},
	}
	events = []*Event{}
)

func main() {
	r := echo.New()

	r.PUT("/user/update/:id", updateUser)
	r.POST("/event/add", addEvent)
	r.GET("/user/:id", getUserByID)
	r.GET("/patch/:event_type/:event_id/:entity_id", getPatchedByEventID)
	r.Start(":8080")
}

func addEvent(c echo.Context) error {
	e := &Event{}
	err := c.Bind(e)
	if err != nil {
		return c.JSON(http.StatusBadRequest, err.Error())
	}

	events = append(events, e)
	return c.JSON(http.StatusCreated, "added")
}

func getUserByID(c echo.Context) error {
	entityID, err := strconv.Atoi(c.Param("entity_id"))
	if err != nil {
		log.Println(err)
		return c.JSON(http.StatusBadRequest, err.Error())
	}

	u, err := getUser(int64(entityID))
	if err != nil {
		return c.JSON(http.StatusOK, err.Error())
	}

	return c.JSON(http.StatusOK, u)
}

func getPatchedByEventID(c echo.Context) error {
	patchType := c.Param("event_type")
	eventID, err := strconv.Atoi(c.Param("event_id"))
	if err != nil {
		log.Println(err)
		return c.JSON(http.StatusBadRequest, err.Error)
	}
	entityID, err := strconv.Atoi(c.Param("entity_id"))
	if err != nil {
		log.Println(err)
		return c.JSON(http.StatusBadRequest, err.Error())
	}
	patched, err := getPatched(patchType, int64(eventID), int64(entityID))
	if err != nil {
		log.Println(err)
		return c.JSON(http.StatusBadRequest, err)
	}

	log.Println(patched)
	return c.JSON(http.StatusOK, patched)
}

func updateUser(c echo.Context) error {
	return c.JSON(http.StatusOK, "updated")
}

func getPatched(patchType string, eventID, entityID int64) (*User, error) {
	u, err := getUser(int64(entityID))
	if err != nil {
		fmt.Println(err)
	}
	requiredEvents, err := getEvents(int64(eventID))
	if err != nil {
		return nil, err
	}

	serialized, err := json.Marshal(u)
	if err != nil {
		return nil, err
	}

	source := make([]byte, 0)
	for i, e := range requiredEvents {
		if i == 0 {
			source = serialized
		}
		source, err = patch(e, patchType, source)
		if err != nil {
			return nil, err
		}
	}

	patched := &User{}
	err = json.Unmarshal(source, patched)
	if err != nil {
		return nil, err
	}

	return patched, nil
}

func patch(e *Event, patchType string, source []byte) ([]byte, error) {
	requiredPatch, err := getRequiredPatch(e, patchType)
	if err != nil {
		return nil, err
	}
	p, err := convertToPatch(requiredPatch)
	if err != nil {
		return nil, err
	}

	patchedAsBytes, err := applyPatch(source, p)
	if err != nil {
		return nil, err
	}

	return patchedAsBytes, nil
}

func getRequiredPatch(e *Event, patchType string) (interface{}, error) {
	var requiredPatch interface{}
	switch patchType {
	case rollback:
		requiredPatch = e.Rollback
	case update:
		requiredPatch = e.Update
	default:
		return nil, errors.New("wrong patch type")
	}

	return requiredPatch, nil
}

func convertToPatch(value interface{}) (jsonpatch.Patch, error) {
	serialized, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	patch, err := jsonpatch.DecodePatch(serialized)
	if err != nil {
		return nil, err
	}

	return patch, nil
}

func getUser(id int64) (*User, error) {
	if u, ok := users[id]; ok {
		return u, nil
	}
	return nil, errors.New("user with this id not exist")
}

func getEvents(id int64) ([]*Event, error) {
	if int(id) <= len(events)-1 {
		return events[int(id)-1:], nil
	}
	return nil, errors.New("event with this id not exist")
}

func applyPatch(entity []byte, patch jsonpatch.Patch) ([]byte, error) {
	patchSerialized, _ := json.Marshal(patch)
	p, err := jsonpatch.DecodePatch(patchSerialized)
	if err != nil {
		return nil, err
	}
	patched, err := p.Apply(entity)
	if err != nil {
		return nil, err
	}
	return patched, err
}
