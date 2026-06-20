// Copyright (c) 2026 NogoChain Contributors
// Use of this source code is governed by an ISC license.

package nogopow

type API struct {
	engine *NogopowEngine
}

func (api *API) Hashrate() float64 {
	return float64(api.engine.HashRate())
}

func (api *API) Mining() bool {
	return api.engine.running
}

func (api *API) GetCacheStats() map[string]interface{} {
	if api.engine.cache != nil {
		return api.engine.cache.Stats()
	}
	return nil
}

func (api *API) GetDifficulty() uint64 {
	return GetMetrics().GetPowSuccess()
}

func (api *API) GetHashRate() uint64 {
	return GetMetrics().GetMatrixOps()
}
