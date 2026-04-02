package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"
)

type companySimulationRequest struct {
	Quarters int `json:"quarters"`
}

func (s *Server) handleCompanyOperations(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/companies/")
	segments := strings.Split(strings.Trim(path, "/"), "/")
	if len(segments) < 2 {
		respondError(w, http.StatusBadRequest, "company id and action required")
		return
	}
	companyID, err := parseID(segments[0])
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid company id")
		return
	}
	action := segments[1]
	switch action {
	case "capital-structure":
		if r.Method != http.MethodGet {
			respondError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		if s.Store == nil {
			respondError(w, http.StatusInternalServerError, "store unavailable")
			return
		}
		structure, ok := s.Store.CompanyCapitalStructure(companyID)
		if !ok {
			respondError(w, http.StatusNotFound, "company not found")
			return
		}
		respondJSON(w, http.StatusOK, structure)
	case "production-status":
		if r.Method != http.MethodGet {
			respondError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		if s.Store == nil {
			respondError(w, http.StatusInternalServerError, "store unavailable")
			return
		}
		status, ok := s.Store.CompanyProductionStatus(companyID)
		if !ok {
			respondError(w, http.StatusNotFound, "company not found")
			return
		}
		respondJSON(w, http.StatusOK, status)
	case "supply-chain":
		if r.Method != http.MethodGet {
			respondError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		if s.Store == nil {
			respondError(w, http.StatusInternalServerError, "store unavailable")
			return
		}
		chain, ok := s.Store.CompanySupplyChain(companyID)
		if !ok {
			respondError(w, http.StatusNotFound, "company not found")
			return
		}
		respondJSON(w, http.StatusOK, chain)
	case "financials":
		if r.Method != http.MethodGet {
			respondError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		if s.Store == nil {
			respondError(w, http.StatusInternalServerError, "store unavailable")
			return
		}
		limit := parseLimit(r, 8)
		respondJSON(w, http.StatusOK, s.Store.CompanyFinancialReports(companyID, limit))
	case "dividends":
		if r.Method != http.MethodGet {
			respondError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		if s.Store == nil {
			respondError(w, http.StatusInternalServerError, "store unavailable")
			return
		}
		limit := parseLimit(r, 8)
		respondJSON(w, http.StatusOK, s.Store.CompanyDividends(companyID, limit))
	case "simulate":
		if r.Method != http.MethodPost {
			respondError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		if s.Store == nil {
			respondError(w, http.StatusInternalServerError, "store unavailable")
			return
		}
		var payload companySimulationRequest
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil && !errors.Is(err, io.EOF) {
			respondError(w, http.StatusBadRequest, "invalid json body")
			return
		}
		quarters := payload.Quarters
		if quarters <= 0 {
			quarters = 1
		}
		var result CompanySimulationResult
		now := time.Now().UTC()
		for i := 0; i < quarters; i++ {
			simulatedAt := now.Add(time.Duration(i) * macroQuarterPeriod)
			runtimeResult, err := s.Store.SimulateCompanyQuarter(companyID, simulatedAt)
			if err != nil {
				respondError(w, http.StatusBadRequest, err.Error())
				return
			}
			result = runtimeResult
		}
		respondJSON(w, http.StatusOK, result)
	case "financing":
		if len(segments) < 3 || segments[2] != "initiate" {
			respondError(w, http.StatusBadRequest, "financing action required")
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
		var payload CompanyFinancingRequest
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil && !errors.Is(err, io.EOF) {
			respondError(w, http.StatusBadRequest, "invalid json body")
			return
		}
		result, err := s.Store.InitiateEquityFinancing(companyID, payload)
		if err != nil {
			respondError(w, http.StatusBadRequest, err.Error())
			return
		}
		respondJSON(w, http.StatusOK, result)
	case "buyback":
		if len(segments) < 3 || segments[2] != "authorize" {
			respondError(w, http.StatusBadRequest, "buyback action required")
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
		var payload CompanyBuybackRequest
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil && !errors.Is(err, io.EOF) {
			respondError(w, http.StatusBadRequest, "invalid json body")
			return
		}
		result, err := s.Store.AuthorizeShareBuyback(companyID, payload)
		if err != nil {
			respondError(w, http.StatusBadRequest, err.Error())
			return
		}
		respondJSON(w, http.StatusOK, result)
	default:
		respondError(w, http.StatusNotFound, "unknown company action")
	}
}
