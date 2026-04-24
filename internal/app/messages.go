package app

type routeMsg struct{ ScreenID string }

type toggleSidebarMsg struct{}

type toggleHideReadMsg struct{}

type quitMsg struct{}

type syncNowMsg struct{}

type openDoctorMsg struct{}

type syncCompletedMsg struct {
	SavedCount   int
	ReadCount    int
	DeletedCount int
	Committed    bool
	Err          error
}
