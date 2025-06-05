package bluraysup

type CompositionState uint32

const (
	CompositionStateNormal CompositionState = iota
	CompositionStateAcquPoint
	CompositionStateEpochStart
	CompositionStateEpochContinue
	CompositionStateInvalid
)
