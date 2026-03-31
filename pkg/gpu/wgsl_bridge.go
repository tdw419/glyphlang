package gpu

func (d *Dispatcher) executeGPU(bytecode []byte, config *Config) ([]Result, error) {
	// For now, default workgroup count to NumVMs / 64
	workgroups := int(config.NumVMs / 64)
	if workgroups == 0 { workgroups = 1 }
	return ExecuteMultiWGSL(string(bytecode), int(config.NumVMs), workgroups)
}
