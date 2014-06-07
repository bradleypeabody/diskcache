diskcache
=========

a simple disk-backed cache in golang

Usage:
------

	cache := NewDiskCache()
	cache.Dir = tmpdir
	cache.CleanupSleep = time.Second * 3
	cache.MaxFiles = 1000    // larger than we'll run into
	cache.MaxBytes = 1 << 20 // 1mb cache
	err = cache.Start()
	// if err ...
	
	err = cache.Set("thekey", []byte("the value"))
	// if err ...
	b, err := cache.Get("thekey")
	// if err ...
