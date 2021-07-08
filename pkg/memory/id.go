package memory

import (
	"fmt"
	"math/rand"
	"strconv"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// generateUniqueId attempts to find a unique id using rand and using the given function to check existence.
// This will attempt a few times before giving up and returning an error.
func generateUniqueId(rand *rand.Rand, exists func(candidate string) bool) (string, error) {
	tries := 10
	for i := 0; i < tries; i++ {
		idCandidate := strconv.Itoa(rand.Int())
		if idCandidate != "" && !exists(idCandidate) {
			return idCandidate, nil
		}
	}
	return "", status.Error(codes.Aborted, fmt.Sprintf("id generation attempts exhausted after %v attempts", tries))
}
