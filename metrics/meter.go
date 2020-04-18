package metrics

// Meter tracks a count of a metric
type Meter struct {
	client Client
	name   string
	tags   []string
}

// NewMeter returns a new meter
func NewMeter(client Client, name string, tagOptions ...TagOption) (*Meter, error) {
	if err := validateMetricName(name); err != nil {
		return nil, err
	}

	return &Meter{
		client: client,
		name:   name,
		tags:   GetTags(tagOptions...),
	}, nil
}

// Count adds the provided value to the metric's count
func (m *Meter) Count(value int64) {
	_ = m.client.Count(m.name, value, m.tags)
}

// Incr adds 1 to the metric's count
func (m *Meter) Incr() {
	_ = m.client.Count(m.name, 1, m.tags)
}

// Decr subtracts 1 from the metric's count
func (m *Meter) Decr() {
	_ = m.client.Count(m.name, -1, m.tags)
}
