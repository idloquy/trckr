package main

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/rodaine/table"
	log "github.com/sirupsen/logrus"

	"github.com/idloquy/trckr/cmd/trckr-cli/config"
	"github.com/idloquy/trckr/cmd/trckr-cli/util"
	"github.com/idloquy/trckr/pkg/api"
	"github.com/idloquy/trckr/pkg/client"
	"github.com/idloquy/trckr/pkg/events"
	"github.com/idloquy/trckr/pkg/tasks"
)

var Version string = "0.0.0"

type TaskCmd interface {
	Tasks() []string
}

type TaskDependentCmd interface {
	TasksToCreate() []string
}

type StartTaskCmd struct {
	TaskName   string `arg:"positional,required"`
	StopReason string `arg:"-s,--stop-reason"`
}

func (cmd *StartTaskCmd) Tasks() []string {
	return []string{cmd.TaskName}
}

func (cmd *StartTaskCmd) TasksToCreate() []string {
	return []string{cmd.TaskName}
}

type StopTaskCmd struct {
	TaskName string `arg:"positional,required"`
}

func (cmd *StopTaskCmd) Tasks() []string {
	return []string{cmd.TaskName}
}

type SwitchTaskCmd struct {
	OldTask string `arg:"positional,required"`
	NewTask string `arg:"positional,required"`
}

func (cmd *SwitchTaskCmd) Tasks() []string {
	return []string{cmd.OldTask, cmd.NewTask}
}

func (cmd *SwitchTaskCmd) TasksToCreate() []string {
	return []string{cmd.NewTask}
}

type RenameTaskCmd struct {
	Task    string `arg:"positional,required"`
	NewName string `arg:"positional,required"`
}

type DeleteTaskCmd struct {
	Task string `arg:"positional,required"`
}

type FixCmd struct {
	Task       *string `arg:"--task" help:"specify the new value for the task (ignored for switch events)"`
	StopReason *string `arg:"--stop-reason" help:"specify the new value for the stop reason (ignored for non-start events)"`
	NewTask    *string `arg:"--new-task" help:"specify the new value for the new task (ignored for non-switch events)"`
}

func (cmd *FixCmd) AnyOptionSpecified() bool {
	return cmd.Task != nil || cmd.StopReason != nil || cmd.NewTask != nil
}

type UndoCmd struct {}

type ListTasksCmd struct {
	Running bool `arg:"-r,--running" help:"only display the running task"`
	Stopped bool `arg:"-s,--stopped" help:"only display stopped tasks"`
}

type HistoryCmd struct {
	TimeRange       string `arg:"-t,--time-range" help:"only display events within the specified time range"`
	NDays           string `arg:"-d,--days" help:"only display events that happened within the specified number of days"`
	DayDividingTask string `arg:"--dividing-task" help:"task to use as the day delimiter"`
	All             bool   `arg:"-a,--all" help:"display the entire history"`
	Raw             bool   `arg:"-r,--raw" help:"use a format that's easier to parse by scripts"`
}

type WatchCmd struct {
	Cmd string `arg:"-c,--command"`
}

type Args struct {
	Verbose    bool           `arg:"-v,--verbose" help:"output debug logs"`
	Quiet      bool           `arg:"-q,--quiet" help:"suppress output"`
	ConfigPath string         `arg:"--config-path" help:"custom path to the config file"`
	Server     string         `arg:"--server" help:"specify the server address, overriding the value from the config file"`
	SkipCreate bool           `arg:"--skip-create" help:"skip pessimistically creating the relevant tasks before performing the requested action"`
	StartTask  *StartTaskCmd  `arg:"subcommand:start-task" help:"start a task"`
	StopTask   *StopTaskCmd   `arg:"subcommand:stop-task" help:"stop a task"`
	SwitchTask *SwitchTaskCmd `arg:"subcommand:switch-task" help:"switch to another task"`
	RenameTask *RenameTaskCmd `arg:"subcommand:rename-task" help:"rename a task"`
	DeleteTask *DeleteTaskCmd `arg:"subcommand:delete-task" help:"delete a task"`
	ListTasks  *ListTasksCmd  `arg:"subcommand:list-tasks" help:"list existing tasks"`
	History    *HistoryCmd    `arg:"subcommand:history" help:"display the task history"`
	Undo       *UndoCmd       `arg:"subcommand:undo" help:"undo the latest event"`
	Fix        *FixCmd        `arg:"subcommand:fix" help:"fix an incorrect operation"`
	Watch      *WatchCmd      `arg:"subcommand:watch" help:"watch for task modifications"`
	// NOTE: We're using this instead of goarg's built-in version reporting
	// feature in order to prevent the version from being printed in help messages.
	Version    bool           `arg:"--version" help:"display the version"`
}

type SubcommandError struct {
	Subcommand string
	Err        error
}

func (e *SubcommandError) Error() string {
	return e.Err.Error()
}

var (
	ErrNoCommandSpecified = errors.New("a command must be specified")
	ErrEmptyTaskName      = errors.New("task names must be non-empty")
)

type CommandError struct {
	Err error
}

func (e *CommandError) Error() string {
	return e.Err.Error()
}

func validateGenericArgs(parser *arg.Parser, args Args) error {
	subcmd := parser.Subcommand()

	switch subcmd := subcmd.(type) {
	case TaskCmd:
		if slices.Contains(subcmd.Tasks(), "") {
			subcmd := parser.SubcommandNames()[0]
			return &SubcommandError{
				Subcommand: subcmd,
				Err:        ErrEmptyTaskName,
			}
		}
	case TaskDependentCmd:
		if slices.Contains(subcmd.TasksToCreate(), "") {
			subcmd := parser.SubcommandNames()[0]
			return &SubcommandError{
				Subcommand: subcmd,
				Err:        ErrEmptyTaskName,
			}
		}
	case *Args:
		return &CommandError{Err: ErrNoCommandSpecified}
	case nil:
		return &CommandError{Err: ErrNoCommandSpecified}
	}

	return nil
}

func parseURLForCommand(rawURL string) (*url.URL, error) {
	parsedURL, err := util.ParseWithScheme(rawURL, "http")

	if err != nil {
		return parsedURL, err
	}

	if parsedURL.Path != "" {
		return parsedURL, fmt.Errorf("a path must not be specified")
	}

	if parsedURL.RawQuery != "" {
		return parsedURL, fmt.Errorf("query params must not be specified")
	}

	return parsedURL, nil
}

type LoggableTaskEvent struct {
	event *api.TaskEvent
}

func (ev *LoggableTaskEvent) DescribeValues() string {
	fields := ev.DescribeValuesFields()
	return util.MapToString(fields)
}

func (ev *LoggableTaskEvent) Describe() string {
	fields := ev.DescribeFields()
	return util.MapToString(fields)
}

func (ev *LoggableTaskEvent) DescribeValuesFields() map[string]any {
	switch taskEv := ev.event.TaskEvent.(type) {
	case events.StartEvent:
		return map[string]any{
			"task":        taskEv.Task,
			"stop_reason": taskEv.StopReason,
		}
	case events.StopEvent:
		return map[string]any{
			"task": taskEv.Task,
		}
	case events.SwitchEvent:
		return map[string]any{
			"old_task": taskEv.OldTask,
			"new_task": taskEv.NewTask,
		}
	default:
		panic(fmt.Sprintf("handling for %s events not implemented", taskEv.Name()))
	}
}

func (ev *LoggableTaskEvent) DescribeFields() map[string]any {
	fields := ev.DescribeValuesFields()
	switch taskEv := ev.event.TaskEvent.(type) {
	case events.StartEvent:
		fields["event_type"] = "start"
	case events.StopEvent:
		fields["event_type"] = "stop"
	case events.SwitchEvent:
		fields["event_type"] = "switch"
	default:
		panic(fmt.Sprintf("handling for %s events not implemented", taskEv.Name()))
	}
	fields["id"] = ev.event.ID
	return fields
}

func formatCrossDayPeriod(per client.FullyBoundedPeriod) string {
	startDayStart := util.GetStartOfDay(per.StartedAt())
	stopDayStart := util.GetStartOfDay(per.StoppedAt())

	if startDayStart.Equal(stopDayStart) {
		panic("period is not a cross-day period")
	}

	var head string
	var foot string

	switch per := per.(type) {
	case client.ProductivePeriod:
		head = fmt.Sprintf("%s %s-\n", per.Task, per.StartTime.In(time.Local).Round(time.Minute).Format("15:04"))
		foot = fmt.Sprintf("%s -%s (%s)", per.Task, per.StopTime.In(time.Local).Round(time.Minute).Format("15:04"), util.FormatDuration(per.Duration()))
	case client.NonProductivePeriod:
		nextDayStart := startDayStart.Add(time.Hour * 24)
		startDayDuration := nextDayStart.Sub(per.StartTime)
		stopDayDuration := per.StopTime.Sub(nextDayStart)

		if per.Reason != "" {
			head = fmt.Sprintf("%s %s\n", util.FormatDuration(startDayDuration), per.Reason)
			foot = fmt.Sprintf("%s %s", util.FormatDuration(stopDayDuration), per.Reason)
		} else {
			head = fmt.Sprintf("%s\n", util.FormatDuration(startDayDuration))
			foot = fmt.Sprintf("%s", util.FormatDuration(stopDayDuration))
		}
	default:
		panic(fmt.Sprintf("handling for fully bounded period type %s not implemented", reflect.TypeOf(per).Name()))
	}

	s := head

	currentDayStart := startDayStart
	for !currentDayStart.Equal(stopDayStart) {
		currentDayStart = currentDayStart.Add(time.Hour * 24)
		s += fmt.Sprintf("|%s\n", currentDayStart.In(time.Local).Format(time.DateOnly))
	}

	s += foot

	return s
}

func formatPeriod(per client.Period, raw bool) string {
	var s string
	switch per := per.(type) {
	case client.PartialProductivePeriod:
		// Note that the `In` call is required to force the local timezone
		// to be used in certain cases.
		per.StartTime = per.StartTime.In(time.Local).Round(time.Minute)
		if !raw {
			s = fmt.Sprintf("%s %s-?", per.Task, per.StartTime.Format("15:04"))
		} else {
			s = fmt.Sprintf("%s %d", per.Task, per.StartTime.Unix())
		}
	case client.ProductivePeriod:
		per.StartTime = per.StartTime.In(time.Local).Round(time.Minute)
		per.StopTime = per.StopTime.In(time.Local).Round(time.Minute)

		if !raw {
			startTimeDayStart := time.Date(per.StartTime.Year(), per.StartTime.Month(), per.StartTime.Day(), 0, 0, 0, 0, time.Local)
			stopTimeDayStart := time.Date(per.StopTime.Year(), per.StopTime.Month(), per.StopTime.Day(), 0, 0, 0, 0, time.Local)

			if !startTimeDayStart.Equal(stopTimeDayStart) {
				s = formatCrossDayPeriod(per)
			} else {
				s = fmt.Sprintf("%s %s-%s (%s)", per.Task, per.StartTime.Format("15:04"), per.StopTime.Format("15:04"), util.FormatDuration(per.Duration()))
			}
		} else {
			s = fmt.Sprintf("%s %d %d", per.Task, per.StartTime.Unix(), per.StopTime.Unix())
		}
	case client.NonProductivePeriod:
		if !raw {
			per.StartTime = per.StartTime.In(time.Local).Round(time.Minute)
			per.StopTime = per.StopTime.In(time.Local).Round(time.Minute)

			startTimeDayStart := time.Date(per.StartTime.Year(), per.StartTime.Month(), per.StartTime.Day(), 0, 0, 0, 0, time.Local)
			stopTimeDayStart := time.Date(per.StopTime.Year(), per.StopTime.Month(), per.StopTime.Day(), 0, 0, 0, 0, time.Local)

			if !startTimeDayStart.Equal(stopTimeDayStart) {
				s = formatCrossDayPeriod(per)
			} else {
				s = util.FormatDuration(per.Duration())
				if per.Reason != "" {
					s += fmt.Sprintf(" %s", per.Reason)
				}
			}
		} else {
			if per.Reason != "" {
				s = fmt.Sprintf("- %s", per.Reason)
			} else {
				s = fmt.Sprintf("-")
			}
		}
	default:
		panic(fmt.Sprintf("handling for period type %s not implemented", reflect.TypeOf(per)))
	}
	return s
}

func main() {
	var args Args
	parserCfg := arg.Config{Program: "trckr-cli"}
	parser, err := arg.NewParser(parserCfg, &args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if len(os.Args) < 1 {
		parser.WriteUsage(os.Stderr)
		os.Exit(2)
	}
	parser.MustParse(os.Args[1:])

	if args.Version {
		fmt.Printf("trckr %s\n", Version)
		return
	}

	if args.Verbose && args.Quiet {
		parser.Fail("only one of --verbose or --quiet must be specified")
	}

	if args.Verbose {
		log.SetLevel(log.DebugLevel)
	}

	if err := validateGenericArgs(parser, args); err != nil {
		var subcmdErr *SubcommandError
		if errors.As(err, &subcmdErr) {
			parser.FailSubcommand(subcmdErr.Error(), subcmdErr.Subcommand)
		} else {
			parser.Fail(err.Error())
		}
		os.Exit(2)
	}

	var cfg config.Config
	if args.ConfigPath != "" {
		cfg, err = config.LoadFromPath(args.ConfigPath)
	} else {
		cfg, err = config.Load()
	}

	if err != nil && !errors.Is(err, config.ErrNoConfigFound) {
		if !args.Quiet {
			fmt.Fprintln(os.Stderr, "error: failed to read config:", err)
		}
		os.Exit(1)
	}

	var rawURL string
	if args.Server != "" {
		rawURL = args.Server
	} else if cfg.Server != "" {
		rawURL = cfg.Server
	} else {
		if !args.Quiet {
			fmt.Fprintln(os.Stderr, "error: no server setting found in the config files or command line arguments")
		}
		os.Exit(1)
	}

	serverURL, err := parseURLForCommand(rawURL)
	if err != nil {
		parser.Fail(fmt.Sprintf("failed to parse server URL: %s", err.Error()))
	}

	c, err := client.New(serverURL)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	if !args.SkipCreate {
		if taskDependentCmd, ok := parser.Subcommand().(TaskDependentCmd); ok {
			for _, task := range taskDependentCmd.TasksToCreate() {
				log.WithFields(log.Fields{"task": task}).Debug("pessimistically creating task...")
				if err := c.CreateTask(task); err != nil {
					if !args.Quiet {
						fmt.Fprintln(os.Stderr, "error: failed to create task:", err)
					}
					os.Exit(1)
				}
			}
		}
	}

	switch {
	case args.StartTask != nil:
		log.WithFields(log.Fields{"task": args.StartTask.TaskName, "stop_reason": args.StartTask.StopReason}).Debug("starting task...")
		if err := c.StartTask(args.StartTask.TaskName, args.StartTask.StopReason); err != nil {
			if !args.Quiet {
				fmt.Fprintln(os.Stderr, "error: failed to start task:", err)
			}
			os.Exit(1)
		}
		if !args.Quiet {
			if args.StartTask.StopReason == "" {
				fmt.Printf("successfully started task %s\n", strconv.Quote(args.StartTask.TaskName))
			} else {
				fmt.Printf("successfully started task %s with stop reason %s\n", strconv.Quote(args.StartTask.TaskName), strconv.Quote(args.StartTask.StopReason))
			}
		}
	case args.StopTask != nil:
		log.WithFields(log.Fields{"task": args.StopTask.TaskName}).Debug("stopping task...")
		if err := c.StopTask(args.StopTask.TaskName); err != nil {
			if !args.Quiet {
				fmt.Fprintln(os.Stderr, "error: failed to stop task:", err)
			}
			os.Exit(1)
		}
		if !args.Quiet {
			fmt.Printf("successfully stopped task %s\n", strconv.Quote(args.StopTask.TaskName))
		}
	case args.SwitchTask != nil:
		log.WithFields(log.Fields{"old_task": args.SwitchTask.OldTask, "new_task": args.SwitchTask.NewTask}).Debug("switching tasks...")
		if err := c.SwitchTask(args.SwitchTask.OldTask, args.SwitchTask.NewTask); err != nil {
			if !args.Quiet {
				fmt.Fprintln(os.Stderr, "error: failed to switch tasks:", err)
			}
			os.Exit(1)
		}
		if !args.Quiet {
			fmt.Printf("successfully switched from task %s to %s\n", strconv.Quote(args.SwitchTask.OldTask), strconv.Quote(args.SwitchTask.NewTask))
		}
	case args.RenameTask != nil:
		log.WithFields(log.Fields{"task": args.RenameTask.Task, "new_name": args.RenameTask.NewName}).Debug("renaming task...")
		if err := c.RenameTask(args.RenameTask.Task, args.RenameTask.NewName); err != nil {
			if !args.Quiet {
				fmt.Fprintln(os.Stderr, "error: failed to rename task:", err)
			}
			os.Exit(1)
		}
		if !args.Quiet {
			fmt.Printf("successfully renamed task %s to %s\n", strconv.Quote(args.RenameTask.Task), strconv.Quote(args.RenameTask.NewName))
		}
	case args.DeleteTask != nil:
		log.WithFields(log.Fields{"task": args.DeleteTask.Task}).Debug("deleting task...")
		if err := c.DeleteTask(args.DeleteTask.Task); err != nil {
			if !args.Quiet {
				fmt.Fprintln(os.Stderr, "error: failed to delete task:", err)
			}
			os.Exit(1)
		}
		if !args.Quiet {
			fmt.Printf("successfully deleted task %s\n", strconv.Quote(args.DeleteTask.Task))
		}
	case args.ListTasks != nil:
		if args.Quiet {
			return
		}

		log.Debug("fetching task list...")
		taskList, err := c.ListTasks()
		if err != nil {
			fmt.Fprintln(os.Stderr, "error: failed to list tasks:", err)
			os.Exit(1)
		}

		if len(taskList) == 0 {
			fmt.Println("no tasks")
			return
		}

		var filteredTaskList []tasks.Task
		if args.ListTasks.Running {
			for _, task := range taskList {
				if task.State == tasks.RunningState {
					filteredTaskList = append(filteredTaskList, task)
					// Only a single task can be running at
					// any given time.
					break
				}
			}
		} else if args.ListTasks.Stopped {
			for _, task := range taskList {
				if task.State == tasks.StoppedState {
					filteredTaskList = append(filteredTaskList, task)
				}
			}
		} else {
			filteredTaskList = taskList
			slices.SortFunc(filteredTaskList, func(task1, task2 tasks.Task) int {
				if task1.State == tasks.RunningState && task2.State == tasks.StoppedState {
					return -1
				} else if task1.State == tasks.StoppedState && task2.State == tasks.RunningState {
					return 1
				} else if (task1.State == tasks.StoppedState && task2.State == tasks.StoppedState) || (task1.State == tasks.RunningState && task2.State == tasks.RunningState) {
					return 0
				}

				if task1.State != tasks.RunningState && task1.State != tasks.StoppedState {
					panic(fmt.Sprintf("handling for task state %s not implemented", strconv.Quote(string(task1.State))))
				}

				panic(fmt.Sprintf("handling for task state %s not implemented", strconv.Quote(string(task2.State))))
			})
		}

		tbl := table.New("TASK", "STATE")
		for _, task := range filteredTaskList {
			tbl.AddRow(task.Name, task.State)
		}
		tbl.Print()
	case args.History != nil:
		var history *client.TaskHistory

		if args.History.All && (args.History.TimeRange != "" || args.History.NDays != "" || args.History.DayDividingTask != "") {
			parser.Fail("-a/--all must not be specified along with other parameters")
		}

		// If not requesting the entire history and neither a time range
		// nor a specific number of days is specified, default to displaying
		// only today's events.
		if !args.History.All && args.History.TimeRange == "" && args.History.NDays == "" {
			args.History.NDays = "1"
		}

		printDates := true

		if args.History.All {
			log.Debug("fetching full history...")
			history, err = c.GetHistory()
			if err != nil {
				if !args.Quiet {
					fmt.Fprintln(os.Stderr, "failed to fetch history:", err)
				}
				os.Exit(1)
			}
			if history.IsEmpty() {
				if !args.Quiet {
					fmt.Println("no events")
				}
				return
			}
		} else if args.History.TimeRange != "" {
			if args.History.NDays != "" {
				parser.Fail("-d/--days can't be specified alongside -t/--time-range")
			}

			timeFields := strings.Split(args.History.TimeRange, "-")
			if len(timeFields) != 2 {
				parser.Fail("invalid time range: two times separated by a '-' character must be specified")
			}

			var timeStart time.Time
			var timeEnd time.Time

			now := time.Now()

			timeStart, err = time.Parse(time.DateTime, timeFields[0])
			if err != nil {
				relativeStart, err := time.Parse(time.TimeOnly, timeFields[0])
				if err != nil {
					parser.Fail(fmt.Sprintf("failed to parse start time: %v", err))
				}
				timeStart = time.Date(now.Year(), now.Month(), now.Day(), relativeStart.Hour(), relativeStart.Minute(), relativeStart.Second(), 0, time.Local)
			}

			timeEnd, err = time.Parse(time.DateTime, timeFields[1])
			if err != nil {
				relativeEnd, err := time.Parse(time.TimeOnly, timeFields[1])
				if err != nil {
					parser.Fail(fmt.Sprintf("failed to parse start time: %v", err))
				}

				timeEnd = time.Date(now.Year(), now.Month(), now.Day(), relativeEnd.Hour(), relativeEnd.Minute(), relativeEnd.Second(), 0, time.Local)
			}

			log.WithFields(log.Fields{"time_start": timeStart, "time_end": timeEnd}).Debug("fetching history...")
			history, err = c.GetHistoryDate(timeStart, timeEnd)
			if err != nil {
				fmt.Fprintln(os.Stderr, "error: failed to fetch history:", err)
				os.Exit(1)
			}
			if history.IsEmpty() {
				if !args.Quiet {
					fmt.Println("no events in the specified range")
				}
				return
			}
		} else {
			// Fetch the history for the specified number of days,
			// delimiting days either by time or by task names, depending
			// on whether a day-dividing task is specified.
			var days int
			days, err = strconv.Atoi(args.History.NDays)
			if err != nil {
				parser.Fail("invalid value for -d/--days: must be an integer")
			}

			if days == 0 {
				parser.Fail("invalid value for -d/--days: must be greater than zero")
			}

			if days == 1 {
				printDates = false
			}

			if args.History.DayDividingTask == "" {
				tomorrow := time.Now().Add(time.Hour * 24)
				dayEnd := time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 0, 0, 0, 0, time.Local)
				timeEnd := dayEnd
				timeStart := timeEnd.Add(-(time.Hour * 24 * time.Duration(days)))
				log.WithFields(log.Fields{"time_start": timeStart, "time_end": timeEnd}).Debug("fetching history...")
				history, err = c.GetHistoryDate(timeStart, timeEnd)
			} else {
				log.WithFields(log.Fields{"days": days, "dividing_task": args.History.DayDividingTask}).Debug("fetching history...")
				history, err = c.GetHistoryDays(days, args.History.DayDividingTask)
			}
			if err != nil {
				if !args.Quiet {
					fmt.Fprintln(os.Stderr, "error: failed to fetch history:", err)
				}
				os.Exit(1)
			}
			if history.IsEmpty() {
				if !args.Quiet {
					if days == 1 {
						fmt.Println("no events today")
					} else {
						fmt.Println("no events in the specified number of days")
					}
				}
				return
			}
		}

		if args.Quiet {
			return
		}

		var splittedHistory []*client.TaskHistory
		if args.History.NDays != "" && args.History.DayDividingTask == "" {
			splittedHistory = history.SplitByDay()
		} else if args.History.DayDividingTask != "" {
			splittedHistory = history.SplitByTask(args.History.DayDividingTask)
		} else {
			splittedHistory = append(splittedHistory, history)
		}

		var prevDay time.Time
		for i, dayHistory := range splittedHistory {
			pers := dayHistory.Periods()
			if len(pers) == 0 {
				continue
			}

			// Note that the `In` call is required to force the local timezone to be used in certain cases.
			firstStart := pers[0].StartedAt().In(time.Local)
			if !args.History.Raw && printDates {
				today := time.Date(firstStart.Year(), firstStart.Month(), firstStart.Day(), 0, 0, 0, 0, time.Local)
				if prevDay.Equal(today) {
					fmt.Printf("--\n\n")
				} else {
					prevDay = today
					fmt.Printf("%s\n\n", firstStart.Format(time.DateOnly))
				}
			}

			for _, per := range pers {
				fmt.Println(formatPeriod(per, args.History.Raw))
			}

			stats := dayHistory.TaskStats()
			if len(stats) > 0 {
				fmt.Println()
				for _, stats := range stats {
					if !args.History.Raw {
						fmt.Printf("%s = %s\n", stats.Task, util.FormatDuration(stats.Duration))
					} else {
						fmt.Printf("%s = %d\n", stats.Task, uint64(stats.Duration.Seconds()))
					}
				}
			}

			if i < len(splittedHistory)-1 {
				fmt.Println()
			}
		}
	case args.Undo != nil:
		log.Debug("fetching latest event...")
		latestEv, err := c.GetLatestEvent()
		if err != nil {
			var appErr *client.ApplicationError
			if errors.As(err, &appErr) && appErr.StatusCode == http.StatusNotFound {
				if !args.Quiet {
					fmt.Fprintln(os.Stderr, "error: no events to undo")
				}
				os.Exit(1)
			}

			if !args.Quiet {
				fmt.Fprintln(os.Stderr, "error: failed to fetch latest event to undo:", err)
			}
			os.Exit(1)
		}

		loggable := &LoggableTaskEvent{&latestEv.Event}
		log.WithFields(loggable.DescribeFields()).Debug("undoing event...")
		if err := c.UndoEvent(latestEv.Event.ID); err != nil {
			if !args.Quiet {
				fmt.Fprintln(os.Stderr, "error: failed to undo event:", err)
			}
			os.Exit(1)
		}

		if !args.Quiet {
			fmt.Println("previous event undone:", loggable.Describe())
		}
	case args.Fix != nil:
		if !args.Fix.AnyOptionSpecified() {
			parser.Fail("at least one modification must be specified")
		}

		log.Debug("fetching latest event...")
		latestEv, err := c.GetLatestEvent()
		if err != nil {
			var appErr *client.ApplicationError
			if errors.As(err, &appErr) && appErr.StatusCode == http.StatusNotFound {
				if !args.Quiet {
					fmt.Fprintln(os.Stderr, "error: no events to fix")
				}
				os.Exit(1)
			}

			if !args.Quiet {
				fmt.Fprintln(os.Stderr, "error: failed to fetch latest event to fix:", err)
			}
			os.Exit(1)
		}

		loggableLatestEv := &LoggableTaskEvent{&latestEv.Event}
		log.WithFields(loggableLatestEv.DescribeFields()).Debug("successfully fetched latest event")

		updatedEv := latestEv.Event
		var potentialNewTask string

		switch transientEv := latestEv.Event.TaskEvent.(type) {
		case events.StartEvent:
			if args.Fix.Task == nil && args.Fix.StopReason == nil {
				parser.Fail("no changes specified for start event: one of --task or --stop-reason is required")
			}

			if args.Fix.Task != nil {
				newTask := *args.Fix.Task
				if newTask == "" {
					parser.Fail("--task should be non-empty for start events")
				}

				transientEv.Task = newTask
				potentialNewTask = newTask
			}
			if args.Fix.StopReason != nil {
				newStopReason := *args.Fix.StopReason

				transientEv.StopReason = newStopReason
			}

			updatedEv.TaskEvent = transientEv
		case events.StopEvent:
			parser.Fail("stop events can't be meaningfully updated")
		case events.SwitchEvent:
			if args.Fix.NewTask == nil {
				parser.Fail("no changes specified for switch event: --new-task is required")
			}

			if args.Fix.NewTask != nil {
				newNewTask := *args.Fix.NewTask
				if newNewTask == "" {
					parser.Fail("--new-task should be non-empty for switch events")
				}

				transientEv.NewTask = newNewTask
				potentialNewTask = newNewTask
			}

			updatedEv.TaskEvent = transientEv
		default:
			panic(fmt.Sprintf("handling for %s events not implemented", transientEv.Name()))
		}
		if latestEv.Event == updatedEv {
			if !args.Quiet {
				fmt.Println("latest event already has the specified values")
			}
			return
		}

		if !args.SkipCreate && potentialNewTask != "" {
			log.WithFields(log.Fields{"task": potentialNewTask}).Debug("pessimistically creating task...")
			if err := c.CreateTask(potentialNewTask); err != nil {
				if !args.Quiet {
					fmt.Fprintln(os.Stderr, "error: failed to create task:", err)
				}
				os.Exit(1)
			}
		}

		loggableUpdatedEv := &LoggableTaskEvent{&updatedEv}
		log.WithFields(loggableUpdatedEv.DescribeFields()).Debug("updating event...")
		err = c.UpdateEvent(latestEv.Event.ID, updatedEv.TaskEvent)
		if err != nil {
			if !args.Quiet {
				fmt.Fprintln(os.Stderr, "error: failed to update event:", err)
			}
			os.Exit(1)
		}

		if !args.Quiet {
			fmt.Printf("successfully modified previous %s event:\n", updatedEv.TaskEvent.Name())
			fmt.Println("old values:", loggableLatestEv.DescribeValues())
			fmt.Println("new values:", loggableUpdatedEv.DescribeValues())
		}
	case args.Watch != nil:
		receiver, err := c.GetEventReceiver()
		if err != nil {
			if !args.Quiet {
				fmt.Fprintln(os.Stderr, "error:", err)
			}
			os.Exit(1)
		}

		for {
			log.Debug("waiting for the next event...")
			ev, err := receiver.Next()
			if err != nil {
				if !args.Quiet {
					fmt.Fprintln(os.Stderr, "error while waiting for events:", err)
				}
				os.Exit(1)
			}

			switch apiEv := ev.Event.Event.(type) {
			case api.TaskEvent:
				evID := strconv.Itoa(apiEv.ID)

				switch ev := apiEv.TaskEvent.(type) {
				case events.StartEvent:
					if !args.Quiet {
						fmt.Printf("task started: id=%d task=%s stop_reason=%s\n", apiEv.ID, strconv.Quote(ev.Task), strconv.Quote(ev.StopReason))
					}
					if args.Watch.Cmd != "" {
						err = exec.Command(args.Watch.Cmd, "start", evID, ev.Task, ev.StopReason).Run()
					}
				case events.StopEvent:
					if !args.Quiet {
						fmt.Printf("task stopped: id=%d task=%s\n", apiEv.ID, strconv.Quote(ev.Task))
					}
					if args.Watch.Cmd != "" {
						err = exec.Command(args.Watch.Cmd, "stop", evID, ev.Task).Run()
					}
				case events.SwitchEvent:
					if !args.Quiet {
						fmt.Printf("task switched: id=%d old_task=%s new_task=%s\n", apiEv.ID, strconv.Quote(ev.OldTask), strconv.Quote(ev.NewTask))
					}
					if args.Watch.Cmd != "" {
						err = exec.Command(args.Watch.Cmd, "switch", evID, ev.OldTask, ev.NewTask).Run()
					}
				default:
					panic(fmt.Sprintf("handling for %s events not implemented", ev.Name()))
				}
			case api.OperationEvent:
				switch ev := apiEv.OperationEvent.(type) {
				case events.CreateTaskEvent:
					if !args.Quiet {
						fmt.Printf("task created: name=%s\n", strconv.Quote(ev.Task.Name))
					}
					if args.Watch.Cmd != "" {
						err = exec.Command(args.Watch.Cmd, "create-task", ev.Task.Name).Run()
					}
				case events.RenameTaskEvent:
					if !args.Quiet {
						fmt.Printf("task renamed: task=%s new_name=%s\n", strconv.Quote(ev.Task), strconv.Quote(ev.NewName))
					}
					if args.Watch.Cmd != "" {
						err = exec.Command(args.Watch.Cmd, "rename-task", ev.Task, ev.NewName).Run()
					}
				case events.DeleteTaskEvent:
					if !args.Quiet {
						fmt.Printf("task deleted: task=%s\n", strconv.Quote(ev.Task))
					}
					if args.Watch.Cmd != "" {
						err = exec.Command(args.Watch.Cmd, "delete-task", ev.Task).Run()
					}
				case events.UndoEvent:
					if !args.Quiet {
						fmt.Println("last event undone")
					}
					if args.Watch.Cmd != "" {
						err = exec.Command(args.Watch.Cmd, "undo").Run()
					}
				default:
					panic(fmt.Sprintf("handling for %s events not implemented", ev.Name()))
				}
			default:
				panic(fmt.Sprintf("handling for events of type %s not implemented", reflect.TypeOf(apiEv).Name()))
			}

			if err != nil {
				var exitErr *exec.ExitError
				if errors.As(err, &exitErr) {
					if !args.Quiet {
						fmt.Fprintln(os.Stderr, "command failed:", string(exitErr.Stderr))
					}
				} else {
					if !args.Quiet {
						fmt.Fprintln(os.Stderr, "failed to execute command:", err)
					}
				}
			}
		}
	}
}
