package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"
)

type missionCompleteRequest struct {
	UserID int64 `json:"user_id"`
}

type contractDeliveryRequest struct {
	UserID   int64 `json:"user_id"`
	Quantity int64 `json:"quantity"`
}

func (s *Server) handleUserRank(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if s.Store == nil {
		respondError(w, http.StatusInternalServerError, "store unavailable")
		return
	}
	userID, status, message := s.resolveUserID(r, parseUserID(r), true)
	if status != 0 {
		respondError(w, status, message)
		return
	}
	info, ok := s.Store.UserRankInfo(userID)
	if !ok {
		respondError(w, http.StatusNotFound, "user not found")
		return
	}
	respondJSON(w, http.StatusOK, info)
}

func (s *Server) handleDailyMissions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if s.Store == nil {
		respondError(w, http.StatusInternalServerError, "store unavailable")
		return
	}
	userID, status, message := s.resolveUserID(r, parseUserID(r), true)
	if status != 0 {
		respondError(w, status, message)
		return
	}
	now := time.Now().UTC()
	statuses, err := s.Store.DailyMissionStatus(userID, now)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, DailyMissionResponse{
		Date:     now.Format("2006-01-02"),
		Missions: statuses,
	})
}

func (s *Server) handleUserMissions(w http.ResponseWriter, r *http.Request) {
	s.handleDailyMissions(w, r)
}

func (s *Server) handleMissionByID(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/missions/")
	path = strings.Trim(path, "/")
	if path == "" || path == "daily" {
		respondError(w, http.StatusNotFound, "mission not found")
		return
	}
	segments := strings.Split(path, "/")
	missionID := segments[0]
	if len(segments) == 1 {
		respondError(w, http.StatusNotFound, "mission action required")
		return
	}
	if segments[1] != "complete" {
		respondError(w, http.StatusNotFound, "unknown mission action")
		return
	}
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if s.Store == nil {
		respondError(w, http.StatusInternalServerError, "store unavailable")
		return
	}
	var payload missionCompleteRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil && !errors.Is(err, io.EOF) {
		respondError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	userID, status, message := s.resolveUserID(r, payload.UserID, true)
	if status != 0 {
		respondError(w, status, message)
		return
	}
	result, err := s.Store.CompleteDailyMission(userID, missionID, time.Now().UTC())
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, result)
}

func (s *Server) handleContracts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if s.Store == nil {
		respondJSON(w, http.StatusOK, []ContractStatus{})
		return
	}
	userID, status, message := s.resolveUserID(r, parseUserID(r), false)
	if status != 0 {
		respondError(w, status, message)
		return
	}
	respondJSON(w, http.StatusOK, s.Store.Contracts(userID))
}

func (s *Server) handleUserContracts(w http.ResponseWriter, r *http.Request) {
	s.handleContracts(w, r)
}

func (s *Server) handleContractByID(w http.ResponseWriter, r *http.Request) {
	contractID, segments, err := parsePathID(r.URL.Path, "/contracts/")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid contract id")
		return
	}
	if s.Store == nil {
		respondError(w, http.StatusInternalServerError, "store unavailable")
		return
	}
	if len(segments) == 0 {
		if r.Method != http.MethodGet {
			respondError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		userID, status, message := s.resolveUserID(r, parseUserID(r), false)
		if status != 0 {
			respondError(w, status, message)
			return
		}
		contract, ok := s.Store.Contract(contractID, userID)
		if !ok {
			respondError(w, http.StatusNotFound, "contract not found")
			return
		}
		respondJSON(w, http.StatusOK, contract)
		return
	}
	switch segments[0] {
	case "deliver":
		if r.Method != http.MethodPost {
			respondError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		var payload contractDeliveryRequest
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			respondError(w, http.StatusBadRequest, "invalid json body")
			return
		}
		userID, status, message := s.resolveUserID(r, payload.UserID, true)
		if status != 0 {
			respondError(w, status, message)
			return
		}
		result, err := s.Store.DeliverContract(userID, contractID, payload.Quantity)
		if err != nil {
			respondError(w, http.StatusBadRequest, err.Error())
			return
		}
		respondJSON(w, http.StatusOK, result)
	default:
		respondError(w, http.StatusNotFound, "unknown contract action")
	}
}
