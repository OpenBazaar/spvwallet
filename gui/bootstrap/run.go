package bootstrap

import (
	"github.com/asticode/go-astilectron"
	"github.com/asticode/go-astilog"
	"github.com/pkg/errors"
	"os"
	"path/filepath"
)

func Asset(src string) ([]byte, error) {
	return []byte{}, nil
}

// Run runs the bootstrap
func Run(o Options) (err error) {
	var a *astilectron.Astilectron
	// Create astilectron
	if a, err = astilectron.New(o.AstilectronOptions); err != nil {
		return errors.Wrap(err, "creating new astilectron failed")
	}
	//a.SetProvisioner(astilectron.NewDisembedderProvisioner(Asset, "../vendor/astilectron-v"+astilectron.VersionAstilectron+".zip", "../vendor/electron-v"+astilectron.VersionElectron+".zip"))
	defer a.Close()
	a.HandleSignals()

	// Adapt astilectron
	if o.AdaptAstilectron != nil {
		o.AdaptAstilectron(a)
	}

	// Base directory path default to executable path
	if o.BaseDirectoryPath == "" {
		if o.BaseDirectoryPath, err = os.Executable(); err != nil {
			return errors.Wrap(err, "getting executable path failed")
		}
		o.BaseDirectoryPath = filepath.Dir(o.BaseDirectoryPath)
	}

	// Provision
	if err = provision(o.BaseDirectoryPath, o.RestoreAssets, o.CustomProvision); err != nil {
		return errors.Wrap(err, "provisioning failed")
	}

	// Start
	if err = a.Start(); err != nil {
		return errors.Wrap(err, "starting astilectron failed")
	}

	// Serve or handle messages
	var url string
	if o.MessageHandler == nil {
		var ln = serve(o.BaseDirectoryPath, o.AdaptRouter, o.TemplateData)
		defer ln.Close()
		url = "http://" + ln.Addr().String() + o.Homepage
	} else {
		url = filepath.Join(o.BaseDirectoryPath, "resources", "app", o.Homepage)
	}

	// Debug
	if o.Debug {
		o.WindowOptions.Width = astilectron.PtrInt(*o.WindowOptions.Width + 700)
	}

	// Init window
	var w *astilectron.Window
	if w, err = a.NewWindow(url, o.WindowOptions); err != nil {
		return errors.Wrap(err, "new window failed")
	}

	// Handle messages
	if o.MessageHandler != nil {
		w.On(astilectron.EventNameWindowEventMessage, handleMessages(w, o.MessageHandler))
	}

	// Create window
	if err = w.Create(); err != nil {
		return errors.Wrap(err, "creating window failed")
	}

	a.On(astilectron.EventNameAppCmdStop, func(e astilectron.Event) (deleteListener bool) {
		close(o.TrayChan)
		o.Wallet.Close()
		return false
	})

	go func() {
		for {
			select {
			case n := <-o.ResizeChan:
				w.Resize(621, n)
			case height := <-o.TransactionChan:
				w.Send(MessageOut{Name: "newTransaction", Payload: height})
			}
		}
	}()

	// Add tray icon
	if o.TrayOptions != nil {
		go func() {
			for range o.TrayChan {
				t := a.NewTray(o.TrayOptions)
				var m = t.NewMenu([]*astilectron.MenuItemOptions{
					{
						Label: astilectron.PtrStr("Open"),
						OnClick: func(e astilectron.Event) (deleteListener bool) {
							go func() {
								w.Show()
								t.Destroy()
							}()

							return false
						},
						Enabled: astilectron.PtrBool(true),
					},
					{
						Label: astilectron.PtrStr("Exit"),
						OnClick: func(e astilectron.Event) (deleteListener bool) {
							a.Stop()
							return false
						},
						Enabled: astilectron.PtrBool(true),
					},
				})

				// Create the menu
				if err = m.Create(); err != nil {
					astilog.Fatal(errors.Wrap(err, "creating tray menu failed"))
				}
				if err = t.Create(); err != nil {
					astilog.Fatal(errors.Wrap(err, "creating tray failed"))
				}
			}
		}()
	}

	// Adapt window
	if o.AdaptWindow != nil {
		o.AdaptWindow(w)
	}

	// Debug
	if o.Debug {
		if err = w.OpenDevTools(); err != nil {
			return errors.Wrap(err, "opening dev tools failed")
		}
	}

	// Blocking pattern
	a.Wait()
	return
}
