package main

type AnimatedProp struct {
	Value  int
	Visual float64
}

func (p *AnimatedProp) UpdateBy(amount float64) {
	if float64(p.Value) == p.Visual {
		return
	}

	//if float64(p.Value)-0.04 <= p.Visual {
	///	p.Visual = float64(p.Value)
	//	return
	//}

	p.Visual += (float64(p.Value) - p.Visual) * amount
}
