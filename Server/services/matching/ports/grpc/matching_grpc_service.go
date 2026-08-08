package grpc

import (
	"context"
	"fmt"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "XTalk/proto/matching"
	"XTalk/services/matching/application/end_match"
	"XTalk/services/matching/application/get_match_history"
	"XTalk/services/matching/application/get_matching_status"
	"XTalk/services/matching/application/join_queue"
	"XTalk/services/matching/application/leave_queue"
	"XTalk/services/matching/domain/entities"
	"XTalk/services/matching/domain/repositories"
	"XTalk/services/matching/domain/valueobjects"
)

// MatchingGRPCService is the gRPC presentation layer
type MatchingGRPCService struct {
	pb.UnimplementedMatchingServiceServer
	joinQueueHandler         *join_queue.Handler
	leaveQueueHandler        *leave_queue.Handler
	endMatchHandler          *end_match.Handler
	getMatchingStatusHandler *get_matching_status.Handler
	getMatchHistoryHandler   *get_match_history.Handler
	queueRepo                repositories.MatchingQueueRepository
}

// NewMatchingGRPCService creates a new MatchingGRPCService
func NewMatchingGRPCService(
	joinQueueHandler *join_queue.Handler,
	leaveQueueHandler *leave_queue.Handler,
	endMatchHandler *end_match.Handler,
	getMatchingStatusHandler *get_matching_status.Handler,
	getMatchHistoryHandler *get_match_history.Handler,
	queueRepo repositories.MatchingQueueRepository,
) *MatchingGRPCService {
	return &MatchingGRPCService{
		joinQueueHandler:         joinQueueHandler,
		leaveQueueHandler:        leaveQueueHandler,
		endMatchHandler:          endMatchHandler,
		getMatchingStatusHandler: getMatchingStatusHandler,
		getMatchHistoryHandler:   getMatchHistoryHandler,
		queueRepo:                queueRepo,
	}
}

// JoinQueue adds a user to the matching queue
func (s *MatchingGRPCService) JoinQueue(ctx context.Context, req *pb.JoinQueueRequest) (*pb.JoinQueueResponse, error) {
	cmd := join_queue.Command{
		UserID:    req.UserId,
		Age:       int(req.Age),
		MinAge:    int(req.MinAge),
		MaxAge:    int(req.MaxAge),
		Interests: req.Interests,
		Gender:    req.Gender,
		Location:  req.Location,
	}

	result, err := s.joinQueueHandler.Handle(ctx, cmd)
	if err != nil {
		return &pb.JoinQueueResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	response := &pb.JoinQueueResponse{
		Success: true,
		Status:  result.Status,
		Message: result.Message,
	}

	if result.Status == "matched" {
		response.MatchId = result.MatchID
		response.MatchedUserId = result.MatchedUserID
		response.ChatId = result.ChatID
		response.MatchScore = float32(result.MatchScore)
	}

	return response, nil
}

// LeaveQueue removes a user from the matching queue
func (s *MatchingGRPCService) LeaveQueue(ctx context.Context, req *pb.LeaveQueueRequest) (*pb.LeaveQueueResponse, error) {
	cmd := leave_queue.Command{
		UserID: req.UserId,
	}

	result, err := s.leaveQueueHandler.Handle(ctx, cmd)
	if err != nil {
		return &pb.LeaveQueueResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &pb.LeaveQueueResponse{
		Success: result.Success,
		Message: result.Message,
	}, nil
}

// GetMatchingStatus retrieves the matching status for a user
func (s *MatchingGRPCService) GetMatchingStatus(ctx context.Context, req *pb.GetMatchingStatusRequest) (*pb.GetMatchingStatusResponse, error) {
	query := get_matching_status.Query{
		UserID: req.UserId,
	}

	result, err := s.getMatchingStatusHandler.Handle(ctx, query)
	if err != nil {
		return &pb.GetMatchingStatusResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	response := &pb.GetMatchingStatusResponse{
		Success: true,
		Status:  result.Status,
		Message: result.Message,
	}

	if result.Status == "in_queue" {
		response.WaitTime = int32(result.WaitTime)
		response.Priority = int32(result.Priority)
	} else if result.Status == "matched" {
		response.MatchId = result.MatchID
		response.ChatId = result.ChatID
		response.MatchedWith = result.MatchedWith
		response.MatchScore = float32(result.MatchScore)
	}

	return response, nil
}

// GetMatchHistory retrieves match history for a user
func (s *MatchingGRPCService) GetMatchHistory(ctx context.Context, req *pb.GetMatchHistoryRequest) (*pb.GetMatchHistoryResponse, error) {
	query := get_match_history.Query{
		UserID: req.UserId,
		Limit:  int(req.Limit),
		Offset: int(req.Offset),
	}

	result, err := s.getMatchHistoryHandler.Handle(ctx, query)
	if err != nil {
		return &pb.GetMatchHistoryResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	matches := make([]*pb.Match, 0, len(result.Matches))
	for _, dto := range result.Matches {
		match := &pb.Match{
			MatchId:     dto.MatchID,
			MatchedWith: dto.MatchedWith,
			ChatId:      dto.ChatID,
			MatchScore:  float32(dto.MatchScore),
			Status:      dto.Status,
			CreatedAt:   timestamppb.New(dto.CreatedAt),
		}

		if dto.CompletedAt != nil {
			match.CompletedAt = timestamppb.New(*dto.CompletedAt)
		}

		matches = append(matches, match)
	}

	return &pb.GetMatchHistoryResponse{
		Success: true,
		Message: "Match history retrieved successfully",
		Matches: matches,
	}, nil
}

// EndMatch ends a match
func (s *MatchingGRPCService) EndMatch(ctx context.Context, req *pb.EndMatchRequest) (*pb.EndMatchResponse, error) {
	cmd := end_match.Command{
		MatchID: req.MatchId,
		UserID:  req.UserId,
	}

	result, err := s.endMatchHandler.Handle(ctx, cmd)
	if err != nil {
		return &pb.EndMatchResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &pb.EndMatchResponse{
		MatchId: result.MatchID,
		Success: result.Success,
		Message: result.Message,
	}, nil
}

// ---------------------  Legacy RPCs  ---------------------
// These delegate to the existing handlers using the preferences-based
// request types from the older API version.

// JoinMatchingQueue is the legacy RPC for joining the queue with preferences.
func (s *MatchingGRPCService) JoinMatchingQueue(ctx context.Context, req *pb.JoinMatchingRequest) (*pb.JoinMatchingResponse, error) {
	prefs := req.GetPreferences()

	var minAge, maxAge int32
	if prefs != nil {
		minAge = prefs.MinAge
		maxAge = prefs.MaxAge
	}
	if minAge == 0 {
		minAge = 18
	}
	if maxAge == 0 {
		maxAge = 99
	}

	var gender string
	if prefs != nil && len(prefs.Genders) > 0 {
		gender = prefs.Genders[0]
	}

	var interests []string
	if prefs != nil {
		interests = prefs.Interests
	}

	cmd := join_queue.Command{
		UserID:    req.UserId,
		Age:       int(minAge), // Legacy RPC has no explicit age field; default to minAge
		MinAge:    int(minAge),
		MaxAge:    int(maxAge),
		Interests: interests,
		Gender:    gender,
	}

	result, err := s.joinQueueHandler.Handle(ctx, cmd)
	if err != nil {
		return &pb.JoinMatchingResponse{Success: false, Message: err.Error()}, nil
	}

	return &pb.JoinMatchingResponse{
		Success: true,
		Message: result.Message,
		QueueId: result.MatchID,
	}, nil
}

// LeaveMatchingQueue is the legacy RPC for leaving the queue.
func (s *MatchingGRPCService) LeaveMatchingQueue(ctx context.Context, req *pb.LeaveMatchingRequest) (*pb.LeaveMatchingResponse, error) {
	cmd := leave_queue.Command{
		UserID: req.UserId,
	}

	result, err := s.leaveQueueHandler.Handle(ctx, cmd)
	if err != nil {
		return &pb.LeaveMatchingResponse{Success: false, Message: err.Error()}, nil
	}

	return &pb.LeaveMatchingResponse{
		Success: result.Success,
		Message: result.Message,
	}, nil
}

// UpdateMatchingPreferences updates a queued user's matching preferences.
func (s *MatchingGRPCService) UpdateMatchingPreferences(ctx context.Context, req *pb.UpdatePreferencesRequest) (*pb.UpdatePreferencesResponse, error) {
	prefs := req.GetPreferences()
	if prefs == nil {
		return &pb.UpdatePreferencesResponse{Success: false, Message: "preferences required"}, nil
	}

	// Fetch existing candidate so we can preserve the age field.
	existing, err := s.queueRepo.FindInQueue(ctx, req.UserId)
	if err != nil {
		return &pb.UpdatePreferencesResponse{Success: false, Message: fmt.Sprintf("user not in queue: %domain", err)}, nil
	}

	minAge := prefs.MinAge
	maxAge := prefs.MaxAge
	if minAge == 0 {
		minAge = int32(existing.Preferences().AgeRange().Min())
	}
	if maxAge == 0 {
		maxAge = int32(existing.Preferences().AgeRange().Max())
	}

	ageRange, err := valueobjects.NewAgeRange(int(minAge), int(maxAge))
	if err != nil {
		return &pb.UpdatePreferencesResponse{Success: false, Message: err.Error()}, nil
	}

	var gender string
	if len(prefs.Genders) > 0 {
		gender = prefs.Genders[0]
	}

	interests := prefs.Interests
	if interests == nil {
		interests = existing.Preferences().Interests()
	}

	newPrefs, err := valueobjects.NewPreferences(
		existing.Preferences().Age(),
		ageRange,
		interests,
		gender,
		existing.Preferences().Location(),
	)
	if err != nil {
		return &pb.UpdatePreferencesResponse{Success: false, Message: err.Error()}, nil
	}

	updated := entities.ReconstructMatchingCandidate(
		req.UserId,
		newPrefs,
		existing.JoinedAt(),
		existing.Priority(),
	)

	if err := s.queueRepo.UpdateCandidate(ctx, updated); err != nil {
		return &pb.UpdatePreferencesResponse{Success: false, Message: err.Error()}, nil
	}

	return &pb.UpdatePreferencesResponse{
		Success: true,
		Message: "Preferences updated",
	}, nil
}
