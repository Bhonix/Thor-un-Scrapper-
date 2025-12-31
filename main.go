package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
	"golang.org/x/net/proxy"
)

const (
	renkSifirla = "\033[0m"
	renkKirmizi = "\033[31m"
	renkYesil   = "\033[32m"
	renkSari    = "\033[33m"
	renkMavi    = "\033[36m"

	tarayiciBasligi = "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:109.0) Gecko/20100101 Firefox/115.0"

	ekranGenisligi  = 1280
	ekranYuksekligi = 800

	maxDosyaAdiUzunlugu = 50
	ayracUzunlugu       = 70

	httpZamanAsimi       = 45 * time.Second
	screenshotZamanAsimi = 30 * time.Second
	portKontrolZamanAsim = 1 * time.Second
	sayfaBeklemeZamani   = 3 * time.Second
)

type TaramaRaporu struct {
	HedefURL       string `json:"hedef_url"`
	Durum          string `json:"durum"`
	HttpKodu       int    `json:"http_kodu"`
	ZamanDamgasi   string `json:"zaman_damgası"`
	HtmlDosyaYolu  string `json:"html_dosyası"`
	EkranGoruntusu string `json:"ekran_görüntüsü,omitempty"`
	HataDetayi     string `json:"hata_detayı,omitempty"`
}

func main() {
	hedefDosyasi := flag.String("hedef", "targets.yaml", "Okunacak hedef listesi dosyası")
	ciktiKlasoru := flag.String("out", "tor_verisi", "Çıktı klasörü adı")
	flag.Parse()

	bannerYazdir()

	if err := os.MkdirAll(*ciktiKlasoru, 0755); err != nil {
		log.Fatalf("%s[HATA] Klasör oluşturulamadı: %v%s\n", renkKirmizi, err, renkSifirla)
	}

	torPortu := torPortuBul()
	fmt.Printf("%s[+] Tor Portu Tespit Edildi: %s%s\n", renkMavi, torPortu, renkSifirla)

	httpIstemci, err := torIstemciOlustur(torPortu)
	if err != nil {
		log.Fatalf("%s[HATA] Tor Proxy bağlantısı kurulamadı: %v%s\n", renkKirmizi, err, renkSifirla)
	}

	fmt.Print("[*] Tor Ağı ve IP Gizliliği Kontrol Ediliyor... ")
	torAktifMi, benimIPm := torDurumuKontrolEt(httpIstemci)
	if torAktifMi {
		fmt.Printf("%s[GÜVENLİ] Tor Aktif! IP: %s%s\n", renkYesil, benimIPm, renkSifirla)
	} else {
		fmt.Printf("%s[RİSK] Tor Ağı Doğrulanamadı! (IP: %s)%s\n", renkKirmizi, benimIPm, renkSifirla)
	}
	fmt.Println(strings.Repeat("-", ayracUzunlugu))

	hedefListesi, err := dosyadanHedefleriOku(*hedefDosyasi)
	if err != nil {
		log.Fatalf("%s[HATA] Hedef dosyası okunamadı: %v%s\n", renkKirmizi, err, renkSifirla)
	}

	fmt.Printf("[*] Toplam %d hedef yüklendi. Tarama başlıyor...\n\n", len(hedefListesi))

	var raporlar []TaramaRaporu
	basariliSayisi := 0
	basarisizSayisi := 0

	for indeks, adres := range hedefListesi {
		func() {
			defer func() {
				if panikMesaji := recover(); panikMesaji != nil {
					fmt.Printf("%s[KRİTİK HATA] Panic yakalandı: %v%s\n", renkKirmizi, panikMesaji, renkSifirla)
					basarisizSayisi++
				}
			}()

			adres = urlDuzenle(adres)
			guvenliDosyaAdi := guvenliDosyaIsmiOlustur(adres)
			zamanDamgasi := time.Now().Format("0201_150405")

			fmt.Printf("[%d/%d] %-35s ", indeks+1, len(hedefListesi), adres)

			rapor := TaramaRaporu{
				HedefURL:     adres,
				ZamanDamgasi: time.Now().Format(time.RFC3339),
			}

			htmlIcerigi, httpDurumKodu, err := htmlCek(httpIstemci, adres)
			rapor.HttpKodu = httpDurumKodu

			if err != nil {

				kisaHataMesaji := hataMesajiKisalt(err)
				fmt.Printf("-> %s[HATA: %s]%s\n", renkKirmizi, kisaHataMesaji, renkSifirla)
				rapor.Durum = "FAIL"
				rapor.HataDetayi = kisaHataMesaji
				basarisizSayisi++
			} else {
				fmt.Printf("-> %s[HTML: OK]%s", renkYesil, renkSifirla)
				rapor.Durum = "SUCCESS"
				basariliSayisi++

				htmlDosyaYolu := filepath.Join(*ciktiKlasoru, fmt.Sprintf("%s_%s.html", guvenliDosyaAdi, zamanDamgasi))
				if err := os.WriteFile(htmlDosyaYolu, []byte(htmlIcerigi), 0644); err != nil {
					log.Printf("UYARI: HTML dosyası yazılamadı: %v", err)
				} else {
					rapor.HtmlDosyaYolu = htmlDosyaYolu
				}

				fmt.Print(" [SS... ")
				ekranGoruntusuYolu := filepath.Join(*ciktiKlasoru, fmt.Sprintf("%s_%s.png", guvenliDosyaAdi, zamanDamgasi))
				ssHatasi := ekranGoruntusuAl(torPortu, adres, ekranGoruntusuYolu)
				if ssHatasi != nil {
					kisaSsHatasi := hataMesajiKisalt(ssHatasi)
					fmt.Printf("%sFAIL: %s%s]\n", renkSari, kisaSsHatasi, renkSifirla)
					if rapor.HataDetayi == "" {
						rapor.HataDetayi = "Screenshot: " + kisaSsHatasi
					} else {
						rapor.HataDetayi += " | Screenshot: " + kisaSsHatasi
					}
				} else {
					fmt.Printf("%sOK%s]\n", renkYesil, renkSifirla)
					rapor.EkranGoruntusu = ekranGoruntusuYolu
				}
			}
			raporlar = append(raporlar, rapor)
		}()
	}

	fmt.Println()
	fmt.Println(strings.Repeat("=", ayracUzunlugu))
	fmt.Printf("%sTARAMA ÖZETİ%s\n", renkMavi, renkSifirla)
	fmt.Println(strings.Repeat("=", ayracUzunlugu))
	fmt.Printf("✓ Başarılı  : %s%d%s\n", renkYesil, basariliSayisi, renkSifirla)
	fmt.Printf("x Başarısız : %s%d%s\n", renkKirmizi, basarisizSayisi, renkSifirla)
	fmt.Printf("+ Toplam    : %d\n", len(hedefListesi))
	fmt.Printf("Kayıtedilen Klasör    : %s\n", *ciktiKlasoru)
	fmt.Println(strings.Repeat("=", ayracUzunlugu))

	if err := raporlariKaydet(*ciktiKlasoru, raporlar); err != nil {
		log.Printf("%sUYARI: Raporlar kaydedilirken hata: %v%s\n", renkSari, err, renkSifirla)
	}

	fmt.Printf("\n%s[✓] Görev Tamamlandı! Raporlar '%s' klasöründe saklandı.%s\n", renkYesil, *ciktiKlasoru, renkSifirla)
}

func torIstemciOlustur(proxyAdresi string) (*http.Client, error) {
	aramaci, err := proxy.SOCKS5("tcp", proxyAdresi, nil, proxy.Direct)
	if err != nil {
		return nil, fmt.Errorf("SOCKS5 dialer oluşturulamadı: %w", err)
	}

	tasima := &http.Transport{
		Dial: aramaci.Dial,
	}

	return &http.Client{
		Transport: tasima,
		Timeout:   httpZamanAsimi,
	}, nil
}

func htmlCek(istemci *http.Client, adres string) (string, int, error) {
	istek, err := http.NewRequest("GET", adres, nil)
	if err != nil {
		return "", 0, fmt.Errorf("istek oluşturulamadı: %w", err)
	}

	istek.Header.Set("User-Agent", tarayiciBasligi)

	cevap, err := istemci.Do(istek)
	if err != nil {
		return "", 0, fmt.Errorf("bağlantı hatası: %w", err)
	}
	defer cevap.Body.Close()

	icerik, err := io.ReadAll(cevap.Body)
	if err != nil {
		return "", cevap.StatusCode, fmt.Errorf("içerik okunamadı: %w", err)
	}

	return string(icerik), cevap.StatusCode, nil
}

func torDurumuKontrolEt(istemci *http.Client) (bool, string) {
	cevap, err := istemci.Get("https://check.torproject.org/api/ip")
	if err != nil {
		return false, "Bağlantı Hatası"
	}
	defer cevap.Body.Close()

	var sonuc struct {
		IsTor bool   `json:"IsTor"`
		IP    string `json:"IP"`
	}

	if err := json.NewDecoder(cevap.Body).Decode(&sonuc); err != nil {
		return false, "Veri Okunamadı"
	}

	return sonuc.IsTor, sonuc.IP
}

func dosyadanHedefleriOku(dosyaYolu string) ([]string, error) {
	dosya, err := os.Open(dosyaYolu)
	if err != nil {
		return nil, fmt.Errorf("dosya açılamadı: %w", err)
	}
	defer dosya.Close()

	var satirlar []string
	tarayici := bufio.NewScanner(dosya)

	for tarayici.Scan() {
		satir := strings.TrimSpace(tarayici.Text())
		satir = strings.TrimPrefix(satir, "- ")
		satir = strings.TrimPrefix(satir, "url: ")
		satir = strings.ReplaceAll(satir, "\"", "")
		satir = strings.ReplaceAll(satir, "'", "")
		if satir != "" && !strings.HasPrefix(satir, "#") {
			satirlar = append(satirlar, satir)
		}
	}

	if err := tarayici.Err(); err != nil {
		return nil, fmt.Errorf("dosya okuma hatası: %w", err)
	}

	return satirlar, nil
}

func raporlariKaydet(klasor string, raporlar []TaramaRaporu) error {
	logDosyasi, err := os.Create(filepath.Join(klasor, "scan_report.log"))
	if err != nil {
		return fmt.Errorf("log dosyası oluşturulamadı: %w", err)
	}
	defer logDosyasi.Close()

	yazici := bufio.NewWriter(logDosyasi)

	yazici.WriteString("╔═══════════════════════════════════════════════════════════════════╗\n")
	yazici.WriteString("║                     TOR TARAMA RAPORU                             ║\n")
	yazici.WriteString("╚═══════════════════════════════════════════════════════════════════╝\n\n")
	yazici.WriteString(fmt.Sprintf("%-25s | %-10s | %-8s | %s\n", "ZAMAN", "DURUM", "HTTP", "HEDEF URL"))
	yazici.WriteString(strings.Repeat("-", ayracUzunlugu) + "\n")

	for _, rapor := range raporlar {
		zamanKisa := rapor.ZamanDamgasi
		if len(zamanKisa) > 19 {
			zamanKisa = zamanKisa[:19]
		}
		yazici.WriteString(fmt.Sprintf("%-25s | %-10s | %-8d | %s\n",
			zamanKisa, rapor.Durum, rapor.HttpKodu, rapor.HedefURL))
	}

	if err := yazici.Flush(); err != nil {
		return fmt.Errorf("log yazma hatası: %w", err)
	}

	jsonDosyasi, err := os.Create(filepath.Join(klasor, "scan_report.json"))
	if err != nil {
		return fmt.Errorf("json dosyası oluşturulamadı: %w", err)
	}
	defer jsonDosyasi.Close()

	kodlayici := json.NewEncoder(jsonDosyasi)
	kodlayici.SetIndent("", "  ")
	if err := kodlayici.Encode(raporlar); err != nil {
		return fmt.Errorf("json yazma hatası: %w", err)
	}

	return nil
}

func torPortuBul() string {
	if portAcikMi("127.0.0.1:9050") {
		return "127.0.0.1:9050"
	}
	if portAcikMi("127.0.0.1:9150") {
		return "127.0.0.1:9150"
	}
	return "127.0.0.1:9050"
}

func portAcikMi(adres string) bool {
	baglanti, err := net.DialTimeout("tcp", adres, portKontrolZamanAsim)
	if err == nil {
		baglanti.Close()
		return true
	}
	return false
}

func ekranGoruntusuAl(proxyAdresi, adres, dosyaYolu string) error {
	secenekler := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.ProxyServer("socks5://"+proxyAdresi),
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("ignore-certificate-errors", true),
		chromedp.WindowSize(ekranGenisligi, ekranYuksekligi),
	)

	baglamYonetici, iptal1 := chromedp.NewExecAllocator(context.Background(), secenekler...)
	defer iptal1()

	baglam, iptal2 := chromedp.NewContext(baglamYonetici)
	defer iptal2()

	baglam, iptal3 := context.WithTimeout(baglam, screenshotZamanAsimi)
	defer iptal3()

	var tampon []byte
	hata := chromedp.Run(baglam,
		chromedp.Navigate(adres),
		chromedp.Sleep(sayfaBeklemeZamani),
		chromedp.CaptureScreenshot(&tampon),
		chromedp.ActionFunc(func(context.Context) error {
			return os.WriteFile(dosyaYolu, tampon, 0644)
		}),
	)
	return hata
}

func urlDuzenle(adres string) string {
	adres = strings.TrimSpace(adres)
	if !strings.HasPrefix(adres, "http") {
		return "http://" + adres
	}
	return adres
}

func guvenliDosyaIsmiOlustur(adres string) string {
	degistirici := strings.NewReplacer(
		"http://", "",
		"https://", "",
		"/", "_",
		":", "",
		".onion", "",
		".", "_",
	)
	isim := degistirici.Replace(adres)

	if len(isim) > maxDosyaAdiUzunlugu {
		return isim[:maxDosyaAdiUzunlugu]
	}
	return isim
}

func hataMesajiKisalt(err error) string {
	mesaj := err.Error()

	if strings.Contains(mesaj, "timeout") || strings.Contains(mesaj, "Timeout") {
		return "ZAMAN AŞIMI"
	}
	if strings.Contains(mesaj, "refused") {
		return "BAĞLANTI REDDEDİLDİ"
	}
	if strings.Contains(mesaj, "no such host") {
		return "HOST BULUNAMADI"
	}
	if strings.Contains(mesaj, "EOF") {
		return "BAĞLANTI KESİLDİ"
	}
	if strings.Contains(mesaj, "certificate") {
		return "SERTİFİKA HATASI"
	}
	if len(mesaj) > 30 {
		return mesaj[:30] + "..."
	}
	return mesaj
}

func bannerYazdir() {
	fmt.Println(renkMavi + `
  _____ _                 _____                                
 |_   _| |               / ____|                               
   | | | |__   ___  _ __| (___   ___ _ __ __ _ _ __   ___ _ __ 
   | | | '_ \ / _ \| '__|\___ \ / __| '__/ _' | '_ \ / _ \ '__|
   | | | | | | (_) | |   ____) | (__| | | (_| | |_) |  __/ |   
   |_| |_| |_|\___/|_|  |_____/ \___|_|  \__,_| .__/ \___|_|   
                                              | |              
   Was Designed BurAkyurek-             |_|              
` + renkSifirla)
	fmt.Println(renkYesil + renkSifirla)
	fmt.Println()
}
