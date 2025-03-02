package list

import (
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
	"testing"

	"gitlab.com/gitlab-org/cli/pkg/iostreams"

	"gitlab.com/gitlab-org/cli/commands/cmdtest"
	"gitlab.com/gitlab-org/cli/commands/issuable"

	"github.com/MakeNowJust/heredoc/v2"

	"github.com/stretchr/testify/assert"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/cli/api"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/config"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
	"gitlab.com/gitlab-org/cli/pkg/httpmock"
	"gitlab.com/gitlab-org/cli/test"
)

func runCommand(command string, rt http.RoundTripper, isTTY bool, cli string, runE func(opts *ListOptions) error, doHyperlinks string) (*test.CmdOut, error) {
	ios, _, stdout, stderr := cmdtest.InitIOStreams(isTTY, doHyperlinks)
	factory := cmdtest.InitFactory(ios, rt)

	// TODO: shouldn't be there but the stub doesn't work without it
	_, _ = factory.HttpClient()

	issueType := issuable.TypeIssue
	if command == "incident" {
		issueType = issuable.TypeIncident
	}

	cmd := NewCmdList(factory, runE, issueType)

	return cmdtest.ExecuteCommand(cmd, cli, stdout, stderr)
}

func TestNewCmdList(t *testing.T) {
	ios, _, _, _ := iostreams.Test()
	ios.IsaTTY = true
	ios.IsInTTY = true
	ios.IsErrTTY = true

	fakeHTTP := httpmock.New()
	defer fakeHTTP.Verify(t)

	factory := &cmdutils.Factory{
		IO: ios,
		HttpClient: func() (*gitlab.Client, error) {
			a, err := api.TestClient(&http.Client{Transport: fakeHTTP}, "", "", false)
			if err != nil {
				return nil, err
			}
			return a.Lab(), err
		},
		Config: func() (config.Config, error) {
			return config.NewBlankConfig(), nil
		},
		BaseRepo: func() (glrepo.Interface, error) {
			return glrepo.New("OWNER", "REPO"), nil
		},
	}
	t.Run("Issue_NewCmdList", func(t *testing.T) {
		gotOpts := &ListOptions{}
		err := NewCmdList(factory, func(opts *ListOptions) error {
			gotOpts = opts
			return nil
		}, issuable.TypeIssue).Execute()

		assert.Nil(t, err)
		assert.Equal(t, factory.IO, gotOpts.IO)

		gotBaseRepo, _ := gotOpts.BaseRepo()
		expectedBaseRepo, _ := factory.BaseRepo()
		assert.Equal(t, gotBaseRepo, expectedBaseRepo)
	})
}

func TestIssueList_tty(t *testing.T) {
	fakeHTTP := httpmock.New()
	defer fakeHTTP.Verify(t)

	fakeHTTP.RegisterResponder(http.MethodGet, "/projects/OWNER/REPO/issues",
		httpmock.NewFileResponse(http.StatusOK, "./testdata/issuableList.json"))

	output, err := runCommand("issue", fakeHTTP, true, "", nil, "")
	if err != nil {
		t.Errorf("error running command `issue list`: %v", err)
	}

	out := output.String()
	timeRE := regexp.MustCompile(`\d+ years`)
	out = timeRE.ReplaceAllString(out, "X years")

	assert.Equal(t, heredoc.Doc(`
		Showing 3 open issues in OWNER/REPO that match your search. (Page 1)

		#6	OWNER/REPO/issues/6	Issue one	(foo, bar) 	about X years ago
		#7	OWNER/REPO/issues/7	Issue two	(fooz, baz)	about X years ago
		#8	OWNER/REPO/issues/8	Incident 	(foo, baz) 	about X years ago

	`), out)
	assert.Equal(t, ``, output.Stderr())
}

func TestIssueList_ids(t *testing.T) {
	fakeHTTP := httpmock.New()
	defer fakeHTTP.Verify(t)

	fakeHTTP.RegisterResponder(http.MethodGet, "/projects/OWNER/REPO/issues",
		httpmock.NewFileResponse(http.StatusOK, "./testdata/issuableList.json"))

	output, err := runCommand("issue", fakeHTTP, true, "-F ids", nil, "")
	if err != nil {
		t.Errorf("error running command `issue list -F ids`: %v", err)
	}

	out := output.String()

	assert.Equal(t, "6\n7\n8\n", out)
	assert.Equal(t, ``, output.Stderr())
}

func TestIssueList_urls(t *testing.T) {
	fakeHTTP := httpmock.New()
	defer fakeHTTP.Verify(t)

	fakeHTTP.RegisterResponder(http.MethodGet, "/projects/OWNER/REPO/issues",
		httpmock.NewFileResponse(http.StatusOK, "./testdata/issuableList.json"))

	output, err := runCommand("issue", fakeHTTP, true, "-F urls", nil, "")
	if err != nil {
		t.Errorf("error running command `issue list -F urls`: %v", err)
	}

	out := output.String()

	assert.Equal(t, heredoc.Doc(`
		http://gitlab.com/OWNER/REPO/issues/6
		http://gitlab.com/OWNER/REPO/issues/7
		http://gitlab.com/OWNER/REPO/issues/8
	`), out)
	assert.Equal(t, ``, output.Stderr())
}

func TestIssueList_tty_withFlags(t *testing.T) {
	t.Run("project", func(t *testing.T) {
		fakeHTTP := httpmock.New()
		defer fakeHTTP.Verify(t)

		fakeHTTP.RegisterResponder(http.MethodGet, "/projects/OWNER/REPO/issues",
			httpmock.NewStringResponse(http.StatusOK, `[]`))

		output, err := runCommand("issue", fakeHTTP, true, "--opened -P1 -p100 --confidential -a someuser -l bug -m1", nil, "")
		if err != nil {
			t.Errorf("error running command `issue list`: %v", err)
		}

		cmdtest.Eq(t, output.Stderr(), "")
		cmdtest.Eq(t, output.String(), `No open issues match your search in OWNER/REPO.


`)
	})
	t.Run("group", func(t *testing.T) {
		fakeHTTP := httpmock.New()
		defer fakeHTTP.Verify(t)

		fakeHTTP.RegisterResponder(http.MethodGet, "/groups/GROUP/issues",
			httpmock.NewStringResponse(http.StatusOK, `[]`))

		output, err := runCommand("issue", fakeHTTP, true, "--group GROUP", nil, "")
		if err != nil {
			t.Errorf("error running command `issue list`: %v", err)
		}

		cmdtest.Eq(t, output.Stderr(), "")
		cmdtest.Eq(t, output.String(), `No open issues match your search in GROUP.


`)
	})
}

func TestIssueList_filterByIteration(t *testing.T) {
	fakeHTTP := &httpmock.Mocker{
		MatchURL: httpmock.PathAndQuerystring,
	}
	defer fakeHTTP.Verify(t)

	fakeHTTP.RegisterResponder(http.MethodGet, "/api/v4/projects/OWNER/REPO/issues?in=title%2Cdescription&iteration_id=9&page=1&per_page=30&state=opened",
		httpmock.NewStringResponse(http.StatusOK, `[]`))

	output, err := runCommand("issue", fakeHTTP, true, "--iteration 9", nil, "")
	if err != nil {
		t.Errorf("error running command `issue list`: %v", err)
	}

	cmdtest.Eq(t, output.Stderr(), "")
	cmdtest.Eq(t, output.String(), `No open issues match your search in OWNER/REPO.


`)
}

func TestIssueList_tty_withIssueType(t *testing.T) {
	fakeHTTP := httpmock.New()
	defer fakeHTTP.Verify(t)

	fakeHTTP.RegisterResponder(http.MethodGet, "/projects/OWNER/REPO/issues",
		httpmock.NewFileResponse(http.StatusOK, "./testdata/incidentList.json"))

	output, err := runCommand("issue", fakeHTTP, true, "--issue-type=incident", nil, "")
	if err != nil {
		t.Errorf("error running command `issue list`: %v", err)
	}

	out := output.String()
	timeRE := regexp.MustCompile(`\d+ years`)
	out = timeRE.ReplaceAllString(out, "X years")

	assert.Equal(t, heredoc.Doc(`
		Showing 1 open incident in OWNER/REPO that match your search. (Page 1)

		#8	OWNER/REPO/issues/8	Incident	(foo, baz)	about X years ago

	`), out)
	assert.Equal(t, ``, output.Stderr())
}

func TestIncidentList_tty_withIssueType(t *testing.T) {
	fakeHTTP := httpmock.New()

	fakeHTTP.RegisterResponder(http.MethodGet, "/projects/OWNER/REPO/issues",
		httpmock.NewFileResponse(http.StatusOK, "./testdata/incidentList.json"))

	output, err := runCommand("incident", fakeHTTP, true, "--issue-type=incident", nil, "")
	if err == nil {
		t.Error("expected an `unknown flag: --issue-type` error, but got nothing")
	}

	assert.Equal(t, ``, output.String())
	assert.Equal(t, ``, output.Stderr())
}

func TestIssueList_tty_mine(t *testing.T) {
	t.Run("mine with all flag and user exists", func(t *testing.T) {
		fakeHTTP := httpmock.New()
		defer fakeHTTP.Verify(t)

		fakeHTTP.RegisterResponder(http.MethodGet, "/projects/OWNER/REPO/issues",
			httpmock.NewStringResponse(http.StatusOK, `[]`))

		fakeHTTP.RegisterResponder(http.MethodGet, "/user",
			httpmock.NewStringResponse(http.StatusOK, `{"username": "john_smith"}`))

		output, err := runCommand("issue", fakeHTTP, true, "--mine -A", nil, "")
		if err != nil {
			t.Errorf("error running command `issue list`: %v", err)
		}

		cmdtest.Eq(t, output.Stderr(), "")
		cmdtest.Eq(t, output.String(), `No issues match your search in OWNER/REPO.


`)
	})
	t.Run("user does not exists", func(t *testing.T) {
		fakeHTTP := httpmock.New()
		defer fakeHTTP.Verify(t)

		fakeHTTP.RegisterResponder(http.MethodGet, "/user",
			httpmock.NewStringResponse(http.StatusNotFound, `{message: 404 Not found}`))

		output, err := runCommand("issue", fakeHTTP, true, "--mine -A", nil, "")
		assert.NotNil(t, err)

		cmdtest.Eq(t, output.Stderr(), "")
		cmdtest.Eq(t, output.String(), "")
	})
}

func makeHyperlink(linkText, targetURL string) string {
	return fmt.Sprintf("\x1b]8;;%s\x1b\\%s\x1b]8;;\x1b\\", targetURL, linkText)
}

func TestIssueList_hyperlinks(t *testing.T) {
	noHyperlinkCells := [][]string{
		{"#6", "OWNER/REPO/issues/6", "Issue one", "(foo, bar)", "about X years ago"},
		{"#7", "OWNER/REPO/issues/7", "Issue two", "(fooz, baz)", "about X years ago"},
		{"#8", "OWNER/REPO/issues/8", "Incident", "(foo, baz)", "about X years ago"},
	}

	hyperlinkCells := [][]string{
		{makeHyperlink("#6", "http://gitlab.com/OWNER/REPO/issues/6"), "OWNER/REPO/issues/6", "Issue one", "(foo, bar)", "about X years ago"},
		{makeHyperlink("#7", "http://gitlab.com/OWNER/REPO/issues/7"), "OWNER/REPO/issues/7", "Issue two", "(fooz, baz)", "about X years ago"},
		{makeHyperlink("#8", "http://gitlab.com/OWNER/REPO/issues/8"), "OWNER/REPO/issues/8", "Incident", "(foo, baz)", "about X years ago"},
	}

	type hyperlinkTest struct {
		forceHyperlinksEnv      string
		displayHyperlinksConfig string
		isTTY                   bool

		expectedCells [][]string
	}

	tests := []hyperlinkTest{
		// FORCE_HYPERLINKS causes hyperlinks to be output, whether or not we're talking to a TTY
		{forceHyperlinksEnv: "1", isTTY: true, expectedCells: hyperlinkCells},
		{forceHyperlinksEnv: "1", isTTY: false, expectedCells: hyperlinkCells},

		// empty/missing display_hyperlinks in config defaults to *not* outputting hyperlinks
		{displayHyperlinksConfig: "", isTTY: true, expectedCells: noHyperlinkCells},
		{displayHyperlinksConfig: "", isTTY: false, expectedCells: noHyperlinkCells},

		// display_hyperlinks: false in config prevents outputting hyperlinks
		{displayHyperlinksConfig: "false", isTTY: true, expectedCells: noHyperlinkCells},
		{displayHyperlinksConfig: "false", isTTY: false, expectedCells: noHyperlinkCells},

		// display_hyperlinks: true in config only outputs hyperlinks if we're talking to a TTY
		{displayHyperlinksConfig: "true", isTTY: true, expectedCells: hyperlinkCells},
		{displayHyperlinksConfig: "true", isTTY: false, expectedCells: noHyperlinkCells},
	}

	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			fakeHTTP := httpmock.New()
			defer fakeHTTP.Verify(t)

			fakeHTTP.RegisterResponder(http.MethodGet, "/projects/OWNER/REPO/issues",
				httpmock.NewFileResponse(http.StatusOK, "./testdata/issuableList.json"))

			doHyperlinks := "never"
			if test.forceHyperlinksEnv == "1" {
				doHyperlinks = "always"
			} else if test.displayHyperlinksConfig == "true" {
				doHyperlinks = "auto"
			}

			output, err := runCommand("issue", fakeHTTP, test.isTTY, "", nil, doHyperlinks)
			if err != nil {
				t.Errorf("error running command `issue list`: %v", err)
			}

			out := output.String()
			timeRE := regexp.MustCompile(`\d+ years`)
			out = timeRE.ReplaceAllString(out, "X years")

			lines := strings.Split(strings.TrimRight(out, "\n"), "\n")

			// first two lines have the header and some separating whitespace, so skip those
			for lineNum, line := range lines[2:] {
				gotCells := strings.Split(line, "\t")
				expectedCells := test.expectedCells[lineNum]

				assert.Equal(t, len(expectedCells), len(gotCells))

				for cellNum, gotCell := range gotCells {
					expectedCell := expectedCells[cellNum]

					assert.Equal(t, expectedCell, strings.Trim(gotCell, " "))
				}
			}
		})
	}
}

func TestIssueListJSON(t *testing.T) {
	fakeHTTP := httpmock.New()
	defer fakeHTTP.Verify(t)

	fakeHTTP.RegisterResponder(http.MethodGet, "/projects/OWNER/REPO/issues",
		httpmock.NewFileResponse(http.StatusOK, "./testdata/issueListFull.json"))

	output, err := runCommand("issue", fakeHTTP, true, "--output json", nil, "")
	if err != nil {
		t.Errorf("error running command `issue list -F json`: %v", err)
	}

	if err != nil {
		panic(err)
	}

	b, err := os.ReadFile("./testdata/issueListFull.json")
	if err != nil {
		fmt.Print(err)
	}

	expectedOut := string(b)

	assert.JSONEq(t, expectedOut, output.String())
	assert.Empty(t, output.Stderr())
}

func TestIssueListMutualOutputFlags(t *testing.T) {
	_, err := runCommand("issue", nil, true, "--output json --output-format ids", nil, "")

	assert.NotNil(t, err)
	assert.EqualError(t, err, "if any flags in the group [output output-format] are set none of the others can be; [output output-format] were all set")
}
