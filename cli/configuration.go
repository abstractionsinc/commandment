package cli

import "unsafe"

// bindConfigurationStorage establishes the exact relationship between one
// command configuration object and one destination storage pointer while both
// types are statically known. Runtime operations reuse only this captured
// relationship; they never rediscover fields by reflection, names, or Go type.
func bindConfigurationStorage[C, T any](configuration *C, storage *T) (configurationBinding, bool) {
	if configuration == nil || storage == nil {
		return configurationBinding{}, false
	}
	base := uintptr(unsafe.Pointer(configuration))
	address := uintptr(unsafe.Pointer(storage))
	configurationSize := unsafe.Sizeof(*configuration)
	valueSize := unsafe.Sizeof(*storage)
	if address < base {
		return configurationBinding{}, false
	}
	offset := address - base
	// A positive-sized configuration owns byte offsets [0,size). Go may place
	// a trailing zero-sized field at an address inside the padded object, but a
	// one-past pointer is never an owned storage location. A zero-sized
	// configuration has only offset zero.
	if configurationSize == 0 {
		if offset != 0 || valueSize != 0 {
			return configurationBinding{}, false
		}
	} else if offset >= configurationSize || valueSize > configurationSize-offset {
		return configurationBinding{}, false
	}
	var zero T
	acceptNil := any(zero) == nil
	read := func(raw any) (any, bool) {
		config, ok := raw.(*C)
		if !ok || config == nil {
			return nil, false
		}
		field := (*T)(unsafe.Add(unsafe.Pointer(config), offset))
		return *field, true
	}
	accept := func(raw, value any) bool {
		config, ok := raw.(*C)
		if !ok || config == nil {
			return false
		}
		if value == nil && acceptNil {
			return true
		}
		_, valueOK := value.(T)
		return valueOK
	}
	write := func(raw, value any) bool {
		config, ok := raw.(*C)
		if !ok || config == nil {
			return false
		}
		typed, valueOK := value.(T)
		if !valueOK {
			if value != nil || !acceptNil {
				return false
			}
			typed = zero
		}
		field := (*T)(unsafe.Add(unsafe.Pointer(config), offset))
		*field = typed
		return true
	}
	binding := acceptLocatedConfigurationBinding(read, accept, write, offset, valueSize)
	return binding, true
}

// configurationBinding is the immutable erased form of one typed destination
// storage relationship established by the public authoring facade. It does not
// discover identity or storage at runtime; it can only read or write the exact
// relationship that was proven while both command configuration and destination
// types were statically known.
type configurationBinding struct {
	read   func(any) (any, bool)
	accept func(any, any) bool
	write  func(any, any) bool
	offset uintptr
	size   uintptr
}

func acceptLocatedConfigurationBinding(read func(any) (any, bool), accept, write func(any, any) bool, offset, size uintptr) configurationBinding {
	if read == nil || write == nil {
		panic("semantic: invalid configuration binding")
	}
	result := configurationBinding{
		read:   read,
		accept: accept,
		write:  write,
		offset: offset,
		size:   size,
	}

	return result
}

func (binding configurationBinding) Valid() bool { return binding.read != nil && binding.write != nil }

func (binding configurationBinding) Read(configuration any) (any, bool) {
	if !binding.Valid() {
		return nil, false
	}
	return binding.read(configuration)
}

func (binding configurationBinding) Accepts(configuration, value any) bool {
	return binding.Valid() && binding.accept != nil && binding.accept(configuration, value)
}

func (binding configurationBinding) Write(configuration, value any) bool {
	return binding.Valid() && binding.write(configuration, value)
}

func (binding configurationBinding) StorageLocation() (uintptr, uintptr, bool) {
	if !binding.Valid() {
		return 0, 0, false
	}
	return binding.offset, binding.size, true
}

func semanticConfigurationBindingsOverlap(left, right configurationBinding) bool {
	leftOffset, leftSize, leftOK := left.StorageLocation()
	rightOffset, rightSize, rightOK := right.StorageLocation()
	if !leftOK || !rightOK {
		return false
	}
	if leftSize == 0 || rightSize == 0 {
		return leftOffset == rightOffset
	}
	return leftOffset < rightOffset+rightSize && rightOffset < leftOffset+leftSize
}

// semanticCommandConfiguration freezes the exact typed configuration allocator for one
// canonical command scope after the public generic relationship is established.
type semanticCommandConfiguration struct {
	new func() any
}

func acceptCommandConfiguration(newConfiguration func() any) semanticCommandConfiguration {
	if newConfiguration == nil {
		panic("semantic: invalid command configuration")
	}
	configuration := semanticCommandConfiguration{
		new: newConfiguration,
	}
	return configuration
}

func (configuration semanticCommandConfiguration) Valid() bool { return configuration.new != nil }

func (configuration semanticCommandConfiguration) New() any {
	if !configuration.Valid() {
		return nil
	}
	return configuration.new()
}
