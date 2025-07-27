package main


import (
    "fmt"
    "log"
    "os"
    "time"

    tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
    "github.com/shirou/gopsutil/v3/cpu"
    "github.com/shirou/gopsutil/v3/disk"
    "github.com/shirou/gopsutil/v3/mem"
    "github.com/shirou/gopsutil/v3/host"
)

var (
    allowedAccounts = map[int64]bool{
        : true,
        : true,
    }
    blockedUsers = make(map[int64]bool)
)
var admin int = 

func getRAMInfo() string {
    v, err := mem.VirtualMemory()
    if err != nil {
        return "Ошибка получения информации об ОЗУ"
    }
    return fmt.Sprintf("Total: %.2f GB\nUsed: %.2f GB\nFree: %.2f GB\nUsedPercent: %.2f%%",
        float64(v.Total)/1e9,
        float64(v.Used)/1e9,
        float64(v.Available)/1e9,
        v.UsedPercent)
}

func getCPUInfo() string {
    loads, err := cpu.Percent(0, false)
    if err != nil || len(loads) == 0 {
        return "Ошибка получения информации о CPU"
    }
    info, err := cpu.Counts(true)
    if err != nil {
        info = 0
    }
    return fmt.Sprintf("Cores: %d\nUsage: %.2f%%", info, loads[0])
}

func getDiskInfo() string {
    usage, err := disk.Usage("/")
    if err != nil {
        return "Ошибка получения информации о диске"
    }
    return fmt.Sprintf("Total: %.2f GB\nUsed: %.2f GB\nFree: %.2f GB\nUsedPercent: %.2f%%",
        float64(usage.Total)/1e9,
        float64(usage.Used)/1e9,
        float64(usage.Free)/1e9,
        usage.UsedPercent)
}

func getSysInfo() string {
    info, err := host.Info()
    if err != nil {
        return "Ошибка получения системной информации"
    }
    return fmt.Sprintf("OS: %s\nPlatform: %s\nPlatform Version: %s\nKernel Version: %s\nHost: %s",
        info.OS, info.Platform, info.PlatformVersion, info.KernelVersion, info.Hostname)
}

func main() {
    token := os.Getenv("TELEGRAM_BOT_TOKEN")
    if token == "" {
        log.Fatal("TELEGRAM_BOT_TOKEN не установлен")
    }

    bot, err := tgbotapi.NewBotAPI(token)
    if err != nil {
        log.Fatal(err)
    }

    u := tgbotapi.NewUpdate(0)
    u.Timeout = 60

    updates := bot.GetUpdatesChan(u)

    for update := range updates {
        if update.Message == nil {
            continue
        }

        userID := update.Message.Chat.ID

        // Проверяем разрешён ли пользователь
        if !allowedAccounts[userID] {
            if !blockedUsers[userID] {
                blockedUsers[userID] = true
                bot.Send(tgbotapi.NewMessage(userID, "⛔ Доступ запрещён"))
                bot.Send(tgbotapi.NewMessage(admin, fmt.Sprintf("Запрос от неизвестного пользователя: %d", userID)))
            }
            continue
        }

        text := update.Message.Text

        switch text {
        case "/start", "/help":
            bot.Send(tgbotapi.NewMessage(userID, "✅ Доступ разрешён. Используй команды /ram /cpu /disk /sysinfo"))
        case "/ram":
            bot.Send(tgbotapi.NewMessage(userID, "💾 ОЗУ:\n"+getRAMInfo()))
        case "/cpu":
            bot.Send(tgbotapi.NewMessage(userID, "🖥️ CPU:\n"+getCPUInfo()))
        case "/disk":
            bot.Send(tgbotapi.NewMessage(userID, "📦 Диск:\n"+getDiskInfo()))
        case "/sysinfo":
            bot.Send(tgbotapi.NewMessage(userID, "🧰 Система:\n"+getSysInfo()))
        default:
            bot.Send(tgbotapi.NewMessage(userID, "Неизвестная команда. Используй /start или /help"))
        }

        // Мониторинг загрузки (можно делать раз в минуту в отдельном горутине)
        v, _ := mem.VirtualMemory()
        c, _ := cpu.Percent(0, false)

        if v.UsedPercent > 86 {
            for id := range allowedAccounts {
                bot.Send(tgbotapi.NewMessage(id, fmt.Sprintf("ОЗУ заполнена на %.2f%% - критично!", v.UsedPercent)))
            }
        }
        if len(c) > 0 && c[0] > 92 {
            for id := range allowedAccounts {
                bot.Send(tgbotapi.NewMessage(id, fmt.Sprintf("Нагрузка на процессор критична: %.2f%%", c[0])))
            }
        }

        // Чтобы не жрать CPU - небольшая задержка
        time.Sleep(500 * time.Millisecond)
    }
}
