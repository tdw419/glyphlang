package gpu

func (d *Dispatcher) executeGPU(bytecode []byte, config *Config) ([]Result, error) {
	workgroups := int(config.NumVMs / 64)
	if workgroups == 0 { workgroups = 1 }
	return ExecuteMultiWGSL(bytecode, int(config.NumVMs), workgroups)
}
