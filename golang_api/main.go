package main

import (

	"work/golang_api/db"
	"work/golang_api/router"
)

// remove space from replay gain
// db content to only include needed columns (json keys) remove the others


func main() {

	db.Init()

	r := router.SetupRouter()

	r.Run("0.0.0.0:8080")
}



//func getLoudnessDistribution(data []Content) ([10]int, error) {
//	var distribution [10]int // Array to hold counts for loudness values 0-9
//
//	for _, item := range data {
//		if streamType, ok := item["stream_2"].(string); !ok || streamType != "music" {
//			continue
//		}
//		loudnessValue, ok := item["loudness_old"].(string)
//		if !ok {
//			continue
//		}
//
//		value, err := strconv.Atoi(loudnessValue)
//		if err != nil {
//			continue
//		}
//
//		if value < 0 || value > 9 {
//			return distribution, fmt.Errorf("loudness value out of range: %d", value)
//		}
//		distribution[value]++
//	}
//
//	return distribution, nil
//}
//
//func stats(c *gin.Context) {
//	contents, err := getContent()
//	if err != nil {
//		c.IndentedJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
//	}
//
//	loudnessDistribution, err := getLoudnessDistribution(contents)
//	if err != nil {
//		log.Printf("Error getting loudness distribution: %v", err)
//		c.IndentedJSON(http.StatusInternalServerError, gin.H{"error": "Failed to get loudness distribution"})
//		return
//	}
//
//	c.HTML(http.StatusOK, "stats.tmpl", gin.H{
//		"title": "Stats",
//		"data":  loudnessDistribution,
//	})
//
//}

// Audio analysis - fix

//func getEnergy(samples []float64) float64 {
//	start := time.Now()
//	// exrtact energy
//	var sum float64
//	for _, sample := range samples {
//		sum += sample * sample
//	}
//
//	averageEnergy := sum / float64(len(samples))
//
//	fmt.Printf("Energy: %.3f\n", averageEnergy)
//	fmt.Printf("Which took: %s seconds\n", time.Since(start).String())
//
//	return averageEnergy
//}
//
//// Find the next power of 2
//func nextPowerOf2(n int) int {
//	if n <= 0 {
//		return 1
//	}
//	return int(math.Pow(2, math.Ceil(math.Log2(float64(n)))))
//}
//
//// Pad array to the next power of 2
//func padToNextPowerOf2(samples []float64) []float64 {
//	paddedSize := nextPowerOf2(len(samples))
//	paddedSamples := make([]float64, paddedSize)
//	copy(paddedSamples, samples)
//	return paddedSamples
//}
//
//func getCentroid(samples []float64, sampleRate float64) float64 {
//	start := time.Now()
//
//    // Pad samples to the next power of 2
//    paddedSamples := padToNextPowerOf2(samples)
//
//	 // Convert samples to complex numbers
//	 complexSamples := gofft.Float64ToComplex128Array(paddedSamples)
//
//	// Perform FFT
//	err := gofft.FFT(complexSamples)
//	if err != nil {
//		fmt.Println("FFT error:", err)
//		return 0
//	}
//	// Debug: Print first few FFT results
//	//for i := 0; i < 10 && i < len(complexSamples); i++ {
//	//	fmt.Printf("FFT[%d]: %v\n", i, complexSamples[i])
//	//}
//    // Calculate magnitudes
//    magnitudes := make([]float64, len(complexSamples))
//    for i, v := range complexSamples {
//        magnitudes[i] = cmplx.Abs(v)
//    }
//
//    // Calculate the spectral centroid
//    var sumMagnitudes, weightedSum float64
//    for i, magnitude := range magnitudes {
//        frequency := float64(i) * float64(sampleRate) / float64(len(paddedSamples))
//        weightedSum += frequency * magnitude
//        sumMagnitudes += magnitude
//    }
//
//	if sumMagnitudes == 0 {
//		fmt.Println("Sum of magnitudes is zero, returning zero centroid")
//		return 0
//	}
//
//    spectralCentroid := weightedSum / sumMagnitudes
//
//	// Debug: Print magnitude and frequency
//	fmt.Printf("SumWeightedMagnitude = %f\n", weightedSum)
//	fmt.Printf("SumMagnitude = %f\n", sumMagnitudes)
//	fmt.Printf("Centroid: %.3f\n", spectralCentroid)
//	fmt.Printf("Which took: %s seconds\n", time.Since(start).String())
//
//	return spectralCentroid
//}
//
//func extractAudioEnergy(f *os.File) (string, string, error) {
//	decoder, err := mp3.NewDecoder(f)
//	if err != nil {
//		return "", "", err
//	}
//	buf := make([]byte, 1024)
//	var samples []float64
//	for {
//		n, err := decoder.Read(buf)
//		if err == io.EOF {
//			break
//		}
//		if err != nil {
//			return "", "", err
//		}
//		for i := 0; i < n; i += 2 {
//			sample := int16(buf[i]) | int16(buf[i+1])<<8
//			samples = append(samples, float64(sample)/32768.0)
//		}
//	}
//
//	sampleRate := float64(decoder.SampleRate())
//
//	averageEnergy := fmt.Sprintf("%.2f", getEnergy(samples))
//	spectralCentroid := fmt.Sprintf("%.2f", getCentroid(samples, sampleRate))
//
//	return averageEnergy, spectralCentroid, nil
//}
