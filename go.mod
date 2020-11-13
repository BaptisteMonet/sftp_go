module main

replace mysftp => ./mysftp

go 1.15

require (
	github.com/go-chi/chi v4.1.2+incompatible
	mysftp v0.0.0-00010101000000-000000000000
)
