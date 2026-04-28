package readers

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
	"labours-go/internal/burndown"
	"labours-go/internal/progress"
)

type YamlReader struct {
	data map[string]interface{}
}

func (r *YamlReader) Read(file io.Reader) error {
	// Initialize progress tracking for YAML reading
	quiet := viper.GetBool("quiet")
	progEstimator := progress.NewProgressEstimator(!quiet)

	progEstimator.StartOperation("Reading YAML data", 1)

	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(&r.data); err != nil {
		progEstimator.FinishOperation()
		return fmt.Errorf("error decoding YAML: %v", err)
	}

	progEstimator.UpdateProgress(1)
	progEstimator.FinishOperation()
	return nil
}

func (r *YamlReader) GetName() string {
	herculesData, ok := r.data["hercules"].(map[string]interface{})
	if !ok {
		return ""
	}
	return herculesData["repository"].(string)
}

func (r *YamlReader) GetHeader() (int64, int64) {
	herculesData, ok := r.data["hercules"].(map[string]interface{})
	if !ok {
		return 0, 0
	}
	begin := int64(herculesData["begin_unix_time"].(int))
	end := int64(herculesData["end_unix_time"].(int))
	return begin, end
}

func (r *YamlReader) GetProjectBurndown() (string, [][]int) {
	burndownData, ok := r.data["Burndown"].(map[string]interface{})
	if !ok {
		return "", nil
	}
	repo := r.GetName()
	matrix := parseBurndownMatrix(burndownData["project"].(string))
	return repo, transposeMatrix(matrix)
}

func (r *YamlReader) GetFilesBurndown() ([]FileBurndown, error) {
	burndownData, ok := r.data["Burndown"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("missing Burndown data in YAML")
	}
	filesData, ok := burndownData["files"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("missing files data in Burndown")
	}

	var fileBurndowns []FileBurndown
	for filename, matrixData := range filesData {
		matrix := parseBurndownMatrix(matrixData.(string))
		fileBurndowns = append(fileBurndowns, FileBurndown{
			Filename: filename,
			Matrix:   transposeMatrix(matrix),
		})
	}
	return fileBurndowns, nil
}

func (r *YamlReader) GetPeopleBurndown() ([]PeopleBurndown, error) {
	burndownData, ok := r.data["Burndown"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("missing Burndown data in YAML")
	}
	peopleData, ok := burndownData["people"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("missing people data in Burndown")
	}

	var peopleBurndowns []PeopleBurndown
	for person, matrixData := range peopleData {
		matrix := parseBurndownMatrix(matrixData.(string))
		peopleBurndowns = append(peopleBurndowns, PeopleBurndown{
			Person: person,
			Matrix: transposeMatrix(matrix),
		})
	}
	return peopleBurndowns, nil
}

func (r *YamlReader) GetOwnershipBurndown() ([]string, map[string][][]int, error) {
	burndownData, ok := r.data["Burndown"].(map[string]interface{})
	if !ok {
		return nil, nil, fmt.Errorf("missing Burndown data in YAML")
	}
	peopleSequence, ok := burndownData["people_sequence"].([]string)
	if !ok {
		return nil, nil, fmt.Errorf("missing people_sequence in Burndown")
	}

	peopleData, ok := burndownData["people"].(map[string]interface{})
	if !ok {
		return nil, nil, fmt.Errorf("missing people data in Burndown")
	}

	ownership := make(map[string][][]int)
	for person, matrixData := range peopleData {
		matrix := parseBurndownMatrix(matrixData.(string))
		ownership[person] = matrix
	}

	return peopleSequence, ownership, nil
}

func (r *YamlReader) GetPeopleInteraction() ([]string, [][]int, error) {
	burndownData, ok := r.data["Burndown"].(map[string]interface{})
	if !ok {
		return nil, nil, fmt.Errorf("missing Burndown data in YAML")
	}
	peopleSequence, ok := burndownData["people_sequence"].([]string)
	if !ok {
		return nil, nil, fmt.Errorf("missing people_sequence in Burndown")
	}
	interactionData, ok := burndownData["people_interaction"].(string)
	if !ok {
		return nil, nil, fmt.Errorf("missing people_interaction data")
	}

	matrix := parseBurndownMatrix(interactionData)
	return peopleSequence, matrix, nil
}

func (r *YamlReader) GetFileCooccurrence() ([]string, [][]int, error) {
	couplesData, ok := r.data["Couples"].(map[string]interface{})
	if !ok {
		return nil, nil, fmt.Errorf("missing Couples data in YAML")
	}

	// Try Python-style nested structure first: files_coocc["index"] and files_coocc["matrix"]
	if filesCoocc, exists := couplesData["files_coocc"].(map[string]interface{}); exists {
		fileIndex, indexOk := filesCoocc["index"].([]string)
		if !indexOk {
			// Try []interface{} and convert to []string
			if indexIntf, ok := filesCoocc["index"].([]interface{}); ok {
				fileIndex = make([]string, len(indexIntf))
				for i, v := range indexIntf {
					if str, ok := v.(string); ok {
						fileIndex[i] = str
					}
				}
				indexOk = true
			}
		}

		if indexOk {
			// Handle both string matrix and map-based sparse matrix format
			if matrixStr, ok := filesCoocc["matrix"].(string); ok {
				// Dense matrix as string
				matrix := parseBurndownMatrix(matrixStr)
				return fileIndex, matrix, nil
			} else if matrixData, ok := filesCoocc["matrix"].([]interface{}); ok {
				// Sparse matrix as array of maps (Python format)
				matrix := parseCoooccurrenceMatrix(matrixData)
				return fileIndex, matrix, nil
			}
		}
	}

	// Fallback to flat structure (original Go format)
	fileIndex, ok := couplesData["file_couples_index"].([]string)
	if !ok {
		return nil, nil, fmt.Errorf("missing file_couples_index in Couples")
	}
	matrixData, ok := couplesData["file_couples_matrix"].(string)
	if !ok {
		return nil, nil, fmt.Errorf("missing file_couples_matrix in Couples")
	}

	matrix := parseBurndownMatrix(matrixData)
	return fileIndex, matrix, nil
}

func (r *YamlReader) GetPeopleCooccurrence() ([]string, [][]int, error) {
	couplesData, ok := r.data["Couples"].(map[string]interface{})
	if !ok {
		return nil, nil, fmt.Errorf("missing Couples data in YAML")
	}

	// Try Python-style nested structure first: people_coocc["index"] and people_coocc["matrix"]
	if peopleCoocc, exists := couplesData["people_coocc"].(map[string]interface{}); exists {
		peopleIndex, indexOk := peopleCoocc["index"].([]string)
		if !indexOk {
			// Try []interface{} and convert to []string
			if indexIntf, ok := peopleCoocc["index"].([]interface{}); ok {
				peopleIndex = make([]string, len(indexIntf))
				for i, v := range indexIntf {
					if str, ok := v.(string); ok {
						peopleIndex[i] = str
					}
				}
				indexOk = true
			}
		}

		if indexOk {
			// Handle both string matrix and map-based sparse matrix format
			if matrixStr, ok := peopleCoocc["matrix"].(string); ok {
				// Dense matrix as string
				matrix := parseBurndownMatrix(matrixStr)
				return peopleIndex, matrix, nil
			} else if matrixData, ok := peopleCoocc["matrix"].([]interface{}); ok {
				// Sparse matrix as array of maps (Python format)
				matrix := parseCoooccurrenceMatrix(matrixData)
				return peopleIndex, matrix, nil
			}
		}
	}

	// Fallback to flat structure (original Go format)
	peopleIndex, ok := couplesData["people_couples_index"].([]string)
	if !ok {
		return nil, nil, fmt.Errorf("missing people_couples_index in Couples")
	}
	matrixData, ok := couplesData["people_couples_matrix"].(string)
	if !ok {
		return nil, nil, fmt.Errorf("missing people_couples_matrix in Couples")
	}

	matrix := parseBurndownMatrix(matrixData)
	return peopleIndex, matrix, nil
}

func (r *YamlReader) GetShotnessCooccurrence() ([]string, [][]int, error) {
	shotnessRecords, err := r.GetShotnessRecords()
	if err != nil {
		return nil, nil, err
	}

	index, matrix := shotnessCounterMatrix(shotnessRecords)
	return index, matrix, nil
}

func shotnessCounterMatrix(records []ShotnessRecord) ([]string, [][]int) {
	index := make([]string, len(records))
	matrix := make([][]int, len(records))
	for i, record := range records {
		index[i] = fmt.Sprintf("%s:%s", record.File, record.Name)
		matrix[i] = make([]int, len(records))
		for tick, count := range record.Counters {
			if tick >= 0 && int(tick) < len(records) {
				matrix[i][int(tick)] = int(count)
			}
		}
	}
	return index, matrix
}

// min32 returns the minimum of two int32 values
func min32(a, b int32) int32 {
	if a < b {
		return a
	}
	return b
}

func (r *YamlReader) GetShotnessRecords() ([]ShotnessRecord, error) {
	shotnessData, ok := r.data["Shotness"].([]interface{})
	if !ok {
		return []ShotnessRecord{}, fmt.Errorf("missing Shotness data in YAML")
	}

	var records []ShotnessRecord
	for _, recordInterface := range shotnessData {
		record, ok := recordInterface.(map[string]interface{})
		if !ok {
			continue // Skip invalid records
		}

		counters := make(map[int32]int32)
		if countData, ok := record["counters"].(map[interface{}]interface{}); ok {
			for timeKey, count := range countData {
				// Convert time key and count to int32
				var timeInt int32
				var countInt int32

				switch t := timeKey.(type) {
				case int:
					timeInt = int32(t)
				case int32:
					timeInt = t
				case int64:
					timeInt = int32(t)
				default:
					continue // Skip invalid time keys
				}

				switch c := count.(type) {
				case int:
					countInt = int32(c)
				case int32:
					countInt = c
				case int64:
					countInt = int32(c)
				default:
					continue // Skip invalid counts
				}

				counters[timeInt] = countInt
			}
		}

		// Safely extract string values
		var recordType, recordName, recordFile string
		if t, ok := record["type"].(string); ok {
			recordType = t
		}
		if n, ok := record["name"].(string); ok {
			recordName = n
		}
		if f, ok := record["file"].(string); ok {
			recordFile = f
		}

		records = append(records, ShotnessRecord{
			Type:     recordType,
			Name:     recordName,
			File:     recordFile,
			Counters: counters,
		})
	}
	return records, nil
}

func (r *YamlReader) GetDeveloperStats() ([]DeveloperStat, error) {
	// This method is deprecated in favor of GetDeveloperTimeSeriesData for Python compatibility
	devData, err := r.GetDeveloperTimeSeriesData()
	if err != nil {
		return nil, err
	}

	// Convert time series data to flat stats (aggregated)
	developerMap := make(map[string]*DeveloperStat)

	for _, dayStats := range devData.Days {
		for devIdx, stats := range dayStats {
			if devIdx < len(devData.People) {
				devName := devData.People[devIdx]
				if existing, exists := developerMap[devName]; exists {
					existing.Commits += stats.Commits
					existing.LinesAdded += stats.LinesAdded
					existing.LinesRemoved += stats.LinesRemoved
					existing.LinesModified += stats.LinesModified
				} else {
					developerMap[devName] = &DeveloperStat{
						Name:          devName,
						Commits:       stats.Commits,
						LinesAdded:    stats.LinesAdded,
						LinesRemoved:  stats.LinesRemoved,
						LinesModified: stats.LinesModified,
					}
				}
			}
		}
	}

	var result []DeveloperStat
	for _, stats := range developerMap {
		result = append(result, *stats)
	}

	return result, nil
}

// GetDeveloperTimeSeriesData returns Python-compatible time series data: (people, days)
// where days is {day: {dev: DevDay}}
func (r *YamlReader) GetDeveloperTimeSeriesData() (*DeveloperTimeSeriesData, error) {
	devsData, ok := r.data["Devs"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("missing Devs data in YAML")
	}

	// Get people list (dev names)
	var people []string
	if peopleData, ok := devsData["people"].([]interface{}); ok {
		for _, p := range peopleData {
			if str, ok := p.(string); ok {
				people = append(people, str)
			}
		}
	} else if devIndex, ok := devsData["dev_index"].([]interface{}); ok {
		for _, p := range devIndex {
			if str, ok := p.(string); ok {
				people = append(people, str)
			}
		}
	} else {
		return nil, fmt.Errorf("missing people/dev_index in Devs")
	}

	// Get ticks (time series data)
	ticks, ok := asMap(devsData["ticks"])
	if !ok {
		return nil, fmt.Errorf("missing ticks in Devs")
	}

	days := make(map[int]map[int]DevDay)

	for dayKey, dayData := range ticks {
		dayInt, ok := convertToInt(dayKey)
		if !ok {
			continue
		}

		dayMap, ok := asMap(dayData)
		if !ok {
			continue
		}

		devs := dayMap
		if nestedDevs, ok := lookupMap(dayMap, "devs"); ok {
			devs = nestedDevs
		}

		dayDevs := make(map[int]DevDay)

		for devKey, devData := range devs {
			devInt, ok := convertToInt(devKey)
			if !ok {
				continue
			}

			devDay, ok := parseYamlDevDay(devData)
			if !ok {
				continue
			}
			dayDevs[devInt] = devDay
		}

		days[dayInt] = dayDevs
	}

	return &DeveloperTimeSeriesData{
		People: people,
		Days:   days,
	}, nil
}

func (r *YamlReader) GetLanguageStats() ([]LanguageStat, error) {
	timeSeries, err := r.GetDeveloperTimeSeriesData()
	if err != nil {
		return nil, fmt.Errorf("failed to get developer time series data: %v", err)
	}
	return aggregateLanguageStats(timeSeries)
}

func (r *YamlReader) GetRuntimeStats() (map[string]float64, error) {
	// Stub: Runtime stats are typically not present in YAML files.
	return nil, fmt.Errorf("runtime stats not implemented for YAML")
}

// Helper function to parse burndown matrices
func parseBurndownMatrix(data string) [][]int {
	lines := strings.Split(data, "\n")
	var matrix [][]int
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		numbers := strings.Fields(line)
		var row []int
		for _, num := range numbers {
			val, err := strconv.Atoi(strings.TrimSpace(num))
			if err != nil {
				continue
			}
			row = append(row, val)
		}
		if len(row) > 0 {
			matrix = append(matrix, row)
		}
	}
	return matrix
}

// parseCoooccurrenceMatrix converts Python's sparse matrix format to dense matrix
// Python format: array of maps where each map represents non-zero values in that row
func parseCoooccurrenceMatrix(data []interface{}) [][]int {
	if len(data) == 0 {
		return [][]int{}
	}

	// Find maximum column index to determine matrix size
	maxCol := 0
	for _, rowData := range data {
		if rowMap, ok := rowData.(map[interface{}]interface{}); ok {
			for colKey := range rowMap {
				if colInt, ok := convertToInt(colKey); ok && colInt > maxCol {
					maxCol = colInt
				}
			}
		}
	}

	// Create dense matrix
	matrix := make([][]int, len(data))
	for i := range matrix {
		matrix[i] = make([]int, maxCol+1)
	}

	// Fill in non-zero values
	for rowIdx, rowData := range data {
		if rowMap, ok := rowData.(map[interface{}]interface{}); ok {
			for colKey, valKey := range rowMap {
				if colInt, ok := convertToInt(colKey); ok {
					if valInt, ok := convertToInt(valKey); ok {
						if colInt < len(matrix[rowIdx]) {
							matrix[rowIdx][colInt] = valInt
						}
					}
				}
			}
		}
	}

	return matrix
}

// convertToInt safely converts various number types to int
func convertToInt(val interface{}) (int, bool) {
	switch v := val.(type) {
	case int:
		return v, true
	case int32:
		return int(v), true
	case int64:
		return int(v), true
	case float64:
		return int(v), true
	case string:
		if i, err := strconv.Atoi(v); err == nil {
			return i, true
		}
	}
	return 0, false
}

func asMap(value interface{}) (map[interface{}]interface{}, bool) {
	switch typed := value.(type) {
	case map[interface{}]interface{}:
		return typed, true
	case map[string]interface{}:
		result := make(map[interface{}]interface{}, len(typed))
		for key, val := range typed {
			result[key] = val
		}
		return result, true
	default:
		return nil, false
	}
}

func lookupMap(values map[interface{}]interface{}, key string) (map[interface{}]interface{}, bool) {
	for candidateKey, value := range values {
		if candidateKey == key {
			return asMap(value)
		}
	}
	return nil, false
}

func parseYamlDevDay(value interface{}) (DevDay, bool) {
	if values, ok := value.([]interface{}); ok {
		return parseYamlDevDayList(values)
	}

	devMap, ok := asMap(value)
	if !ok {
		return DevDay{}, false
	}

	var day DevDay
	if commits, ok := lookupInt(devMap, "commits"); ok {
		day.Commits = commits
	}
	if added, ok := lookupInt(devMap, "added"); ok {
		day.LinesAdded = added
	}
	if removed, ok := lookupInt(devMap, "removed"); ok {
		day.LinesRemoved = removed
	}
	if changed, ok := lookupInt(devMap, "changed"); ok {
		day.LinesModified = changed
	}
	if languages, ok := lookupLanguages(devMap, "languages"); ok {
		day.Languages = languages
	}
	return day, true
}

func parseYamlDevDayList(values []interface{}) (DevDay, bool) {
	if len(values) < 4 {
		return DevDay{}, false
	}

	var day DevDay
	if commits, ok := convertToInt(values[0]); ok {
		day.Commits = commits
	}
	if added, ok := convertToInt(values[1]); ok {
		day.LinesAdded = added
	}
	if removed, ok := convertToInt(values[2]); ok {
		day.LinesRemoved = removed
	}
	if changed, ok := convertToInt(values[3]); ok {
		day.LinesModified = changed
	}
	if len(values) >= 5 {
		if languages, ok := parseYamlLanguages(values[4]); ok {
			day.Languages = languages
		}
	}
	return day, true
}

func lookupInt(values map[interface{}]interface{}, key string) (int, bool) {
	for candidateKey, value := range values {
		if candidateKey == key {
			return convertToInt(value)
		}
	}
	return 0, false
}

func lookupLanguages(values map[interface{}]interface{}, key string) (map[string][]int, bool) {
	for candidateKey, value := range values {
		if candidateKey == key {
			return parseYamlLanguages(value)
		}
	}
	return nil, false
}

func parseYamlLanguages(value interface{}) (map[string][]int, bool) {
	languageMap, ok := asMap(value)
	if !ok {
		return nil, false
	}

	languages := make(map[string][]int)
	for languageKey, statsValue := range languageMap {
		language, ok := languageKey.(string)
		if !ok {
			continue
		}
		stats, ok := parseYamlIntList(statsValue)
		if !ok || len(stats) < 3 {
			continue
		}
		languages[language] = stats[:3]
	}
	return languages, true
}

func parseYamlIntList(value interface{}) ([]int, bool) {
	rawValues, ok := value.([]interface{})
	if !ok {
		return nil, false
	}

	values := make([]int, 0, len(rawValues))
	for _, rawValue := range rawValues {
		if intValue, ok := convertToInt(rawValue); ok {
			values = append(values, intValue)
		}
	}
	return values, true
}

// GetBurndownParameters retrieves burndown parameters for YAML reader
func (r *YamlReader) GetBurndownParameters() (burndown.BurndownParameters, error) {
	burndownData, ok := r.data["Burndown"].(map[string]interface{})
	if !ok {
		return burndown.BurndownParameters{}, fmt.Errorf("missing Burndown data in YAML")
	}

	// Extract parameters from YAML - these ARE present in hercules YAML output
	var sampling int = 1         // Default
	var granularity int = 1      // Default
	var tickSize float64 = 86400 // Default (24 hours)

	if val, exists := burndownData["sampling"]; exists {
		if intVal, ok := val.(int); ok {
			sampling = intVal
		}
	}

	if val, exists := burndownData["granularity"]; exists {
		if intVal, ok := val.(int); ok {
			granularity = intVal
		}
	}

	if val, exists := burndownData["tick_size"]; exists {
		if intVal, ok := val.(int); ok {
			tickSize = float64(intVal)
		} else if floatVal, ok := val.(float64); ok {
			tickSize = floatVal
		}
	}

	return burndown.BurndownParameters{
		Sampling:    sampling,
		Granularity: granularity,
		TickSize:    tickSize,
	}, nil
}

// GetProjectBurndownWithHeader retrieves project burndown with header for YAML reader
func (r *YamlReader) GetProjectBurndownWithHeader() (burndown.BurndownHeader, string, [][]int, error) {
	// Get the basic data
	name, matrix := r.GetProjectBurndown()
	if len(matrix) == 0 {
		return burndown.BurndownHeader{}, "", nil, fmt.Errorf("no project burndown data")
	}

	// Get header info
	start, last := r.GetHeader()
	params, _ := r.GetBurndownParameters()

	header := burndown.BurndownHeader{
		Start:       start,
		Last:        last,
		Sampling:    params.Sampling,
		Granularity: params.Granularity,
		TickSize:    params.TickSize,
	}

	return header, name, matrix, nil
}
