package kafka

import (
	"context"
	"github.com/twmb/franz-go/pkg/kmsg"
)

func (s *Service) ListPartitionReassignments(ctx context.Context) (*kmsg.ListPartitionReassignmentsResponse, error) {
	// TODO: Check if target Kafka cluster supports this request
	req := kmsg.NewListPartitionReassignmentsRequest()
	req.Topics = nil // List for all topics

	return req.RequestWith(ctx, s.KafkaClient)
}

func (s *Service) AlterPartitionAssignments(ctx context.Context, topics []kmsg.AlterPartitionAssignmentsRequestTopic) (*kmsg.AlterPartitionAssignmentsResponse, error) {
	// TODO: Check if target Kafka cluster supports this request
	req := kmsg.NewAlterPartitionAssignmentsRequest()
	req.Topics = topics

	return req.RequestWith(ctx, s.KafkaClient)
}
