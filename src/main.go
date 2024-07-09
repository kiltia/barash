package main

func main() {
	config := NewRunnerConfig()
	if config == nil {
		return
	}
	runner := NewRunner[VerificationResult, VerificationResponse](
		*config,
	)
	if runner == nil {
		return
	}
	runner.Run()
}
