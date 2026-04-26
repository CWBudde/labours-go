package readers

import (
	"bytes"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"labours-go/internal/pb"
)

func TestProtobufReader_CurrentHerculesReportPayloads(t *testing.T) {
	contents := map[string][]byte{
		"Burndown": marshalProto(t, &pb.BurndownAnalysisResults{
			Project: &pb.BurndownSparseMatrix{
				Name:            "project",
				NumberOfRows:    1,
				NumberOfColumns: 2,
				Rows: []*pb.BurndownSparseMatrixRow{
					{Columns: []uint32{10, 8}},
				},
			},
			RepositorySequence: []string{"repo-a", "repo-b"},
			Repositories: []*pb.BurndownSparseMatrix{
				{
					Name:            "repo-a",
					NumberOfRows:    1,
					NumberOfColumns: 2,
					Rows: []*pb.BurndownSparseMatrixRow{
						{Columns: []uint32{7, 5}},
					},
				},
			},
		}),
		"Sentiment": marshalProto(t, &pb.CommentSentimentResults{
			SentimentByTick: map[int32]*pb.Sentiment{
				3: {
					Value:    0.75,
					Comments: []string{"looks good"},
					Commits:  []string{"abc123"},
				},
			},
		}),
		"TemporalActivity": marshalProto(t, &pb.TemporalActivityResults{
			Activities: map[int32]*pb.DeveloperTemporalActivity{
				0: {
					Weekdays: &pb.TemporalDimension{Commits: []int32{1, 2}, Lines: []int32{10, 20}},
					Hours:    &pb.TemporalDimension{Commits: []int32{3}, Lines: []int32{30}},
					Months:   &pb.TemporalDimension{Commits: []int32{4}, Lines: []int32{40}},
					Weeks:    &pb.TemporalDimension{Commits: []int32{5}, Lines: []int32{50}},
				},
			},
			DevIndex: []string{"dev-a"},
			Ticks: map[int32]*pb.TemporalActivityTickDevs{
				1: {
					Devs: map[int32]*pb.TemporalActivityTick{
						0: {Commits: 2, Lines: 12, Weekday: 1, Hour: 9, Month: 4, Week: 17},
					},
				},
			},
			TickSize: 86400,
		}),
		"BusFactor": marshalProto(t, &pb.BusFactorAnalysisResults{
			Snapshots: map[int32]*pb.BusFactorTickSnapshot{
				1: {BusFactor: 2, TotalLines: 100, AuthorLines: map[int32]int64{0: 60, 1: 40}},
			},
			SubsystemBusFactor: map[string]int32{"cmd": 1},
			DevIndex:           []string{"dev-a", "dev-b"},
			TickSize:           86400,
			Threshold:          0.8,
		}),
		"OwnershipConcentration": marshalProto(t, &pb.OwnershipConcentrationResults{
			Snapshots: map[int32]*pb.OwnershipConcentrationTickSnapshot{
				1: {Gini: 0.3, Hhi: 0.6, TotalLines: 100, AuthorLines: map[int32]int64{0: 60, 1: 40}},
			},
			SubsystemGini: map[string]float64{"cmd": 0.2},
			SubsystemHhi:  map[string]float64{"cmd": 0.5},
			DevIndex:      []string{"dev-a", "dev-b"},
			TickSize:      86400,
		}),
		"KnowledgeDiffusion": marshalProto(t, &pb.KnowledgeDiffusionResults{
			Files: map[string]*pb.KnowledgeDiffusionFileData{
				"main.go": {
					UniqueEditorsCount:    2,
					UniqueEditorsOverTime: map[int32]int32{1: 1, 2: 2},
					RecentEditorsCount:    1,
					Authors:               []int32{0, 1},
				},
			},
			Distribution: map[int32]int32{2: 1},
			WindowMonths: 6,
			DevIndex:     []string{"dev-a", "dev-b"},
			TickSize:     86400,
		}),
		"HotspotRisk": marshalProto(t, &pb.HotspotRiskResults{
			WindowDays: 90,
			Files: []*pb.FileRisk{
				{
					Path:                "main.go",
					RiskScore:           0.9,
					Size:                100,
					Churn:               12,
					CouplingDegree:      3,
					OwnershipGini:       0.4,
					SizeNormalized:      0.5,
					ChurnNormalized:     0.6,
					CouplingNormalized:  0.7,
					OwnershipNormalized: 0.8,
				},
			},
		}),
		"RefactoringProxy": marshalProto(t, &pb.RefactoringProxyResults{
			Ticks:         []int32{1},
			RenameRatios:  []float32{0.4},
			IsRefactoring: []bool{true},
			TotalChanges:  []int32{10},
			Threshold:     0.3,
			TickSize:      int64(86400 * 1_000_000_000),
		}),
	}

	payload := &pb.AnalysisResults{
		Header: &pb.Metadata{
			Repository:    "repo",
			BeginUnixTime: 1700000000,
			EndUnixTime:   1700864000,
		},
		Contents: contents,
	}

	reader := &ProtobufReader{}
	require.NoError(t, reader.Read(bytes.NewReader(marshalProto(t, payload))))

	repos, err := reader.GetRepositoriesBurndown()
	require.NoError(t, err)
	require.Len(t, repos, 1)
	require.Equal(t, "repo-a", repos[0].Repository)

	names, err := reader.GetRepositoryNames()
	require.NoError(t, err)
	require.Equal(t, []string{"repo-a", "repo-b"}, names)

	sentiment, err := reader.GetSentimentByTick()
	require.NoError(t, err)
	require.Equal(t, float32(0.75), sentiment[3].Value)

	temporal, err := reader.GetTemporalActivity()
	require.NoError(t, err)
	require.Equal(t, []string{"dev-a"}, temporal.People)
	require.Equal(t, 2, temporal.Ticks[1][0].Commits)

	busFactor, err := reader.GetBusFactor()
	require.NoError(t, err)
	require.Equal(t, 2, busFactor.Snapshots[1].BusFactor)
	require.Equal(t, 1, busFactor.SubsystemBusFactor["cmd"])

	ownership, err := reader.GetOwnershipConcentration()
	require.NoError(t, err)
	require.Equal(t, 0.3, ownership.Snapshots[1].Gini)
	require.Equal(t, 0.5, ownership.SubsystemHHI["cmd"])

	diffusion, err := reader.GetKnowledgeDiffusion()
	require.NoError(t, err)
	require.Equal(t, 2, diffusion.Files["main.go"].UniqueEditors)
	require.Equal(t, 1, diffusion.Distribution[2])

	hotspot, err := reader.GetHotspotRisk()
	require.NoError(t, err)
	require.Equal(t, 90, hotspot.WindowDays)
	require.Equal(t, "main.go", hotspot.Files[0].Path)

	refactoring, err := reader.GetRefactoringProxy()
	require.NoError(t, err)
	require.Equal(t, float32(0.4), refactoring.Ticks[0].RefactoringRate)
	require.True(t, refactoring.Ticks[0].IsRefactoring)
}

func TestProtobufReader_ReportPayloadErrorsAreTyped(t *testing.T) {
	t.Run("missing", func(t *testing.T) {
		reader := &ProtobufReader{}
		payload := &pb.AnalysisResults{
			Header:   &pb.Metadata{Repository: "repo"},
			Contents: map[string][]byte{},
		}
		require.NoError(t, reader.Read(bytes.NewReader(marshalProto(t, payload))))

		_, err := reader.GetTemporalActivity()
		require.Error(t, err)
		require.True(t, errors.Is(err, ErrAnalysisMissing))
	})

	t.Run("malformed", func(t *testing.T) {
		reader := &ProtobufReader{}
		payload := &pb.AnalysisResults{
			Header: &pb.Metadata{Repository: "repo"},
			Contents: map[string][]byte{
				"TemporalActivity": []byte("not protobuf"),
			},
		}
		require.NoError(t, reader.Read(bytes.NewReader(marshalProto(t, payload))))

		_, err := reader.GetTemporalActivity()
		require.Error(t, err)
		require.True(t, errors.Is(err, ErrAnalysisMalformed))
	})
}

func marshalProto(t *testing.T, message proto.Message) []byte {
	t.Helper()
	data, err := proto.Marshal(message)
	require.NoError(t, err)
	return data
}
