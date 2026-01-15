import Cocoa

class AppDelegate: NSObject, NSApplicationDelegate {

    var statusItem: NSStatusItem!
    var bridgeProcess: Process?

    func applicationDidFinishLaunching(_ notification: Notification) {
        setupMenu()
        startBridge()
    }

    func setupMenu() {
        statusItem = NSStatusBar.system.statusItem(withLength: NSStatusItem.squareLength)

        if let button = statusItem.button {
            let image = NSImage(named: "StatusIcon")  // ← Resources 内の画像名
            image?.isTemplate = true  // ダークモード対応
            button.image = image
        }

        let menu = NSMenu()
        menu.addItem(
            NSMenuItem(
                title: "設定を開く",
                action: #selector(openSettings),
                keyEquivalent: ","
            ))
        menu.addItem(NSMenuItem.separator())
        menu.addItem(
            NSMenuItem(
                title: "再起動",
                action: #selector(restartBridge),
                keyEquivalent: "r"
            ))
        menu.addItem(NSMenuItem.separator())
        menu.addItem(
            NSMenuItem(
                title: "HAMLAB Bridge 終了",
                action: #selector(quit),
                keyEquivalent: "q"
            ))
        statusItem.menu = menu
    }

    func startBridge() {
        let proc = Process()
        proc.executableURL = URL(
            fileURLWithPath:
                Bundle.main.bundlePath + "/Contents/MacOS/hamlab-bridge"
        )

        do {
            try proc.run()
            bridgeProcess = proc
        } catch {
            NSLog("Failed to start hamlab-bridge: \(error)")
        }
    }

    @objc func openSettings() {
        if let url = URL(string: "http://127.0.0.1:17801/settings") {
            NSWorkspace.shared.open(url)
        }
    }

    @objc func restartBridge() {
        bridgeProcess?.terminate()
        bridgeProcess?.waitUntilExit()
        startBridge()
    }

    @objc func quit() {
        bridgeProcess?.terminate()
        NSApp.terminate(nil)
    }
}
