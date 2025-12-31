ThorScraper, siber tehdit istihbaratı (CTI) araştırmacıları için geliştirilmiş, Tor ağı üzerinden anonim olarak .onion adreslerini tarayan otomatize bir Go uygulamasıdır. Onlarda Dark Web sitesini tek komutla tarayarak HTML içeriklerini ve ekran görüntülerini toplar.



Güvenlik ve Anonimlik

Tor Entegrasyonu - Tüm trafik SOCKS5 proxy üzerinden
IP Sızıntısı Önleme - Özel HTTP Transport yapılandırması
User-Agent Spoofing - Gerçek tarayıcı simülasyonu
Tor Doğrulama - check.torproject.org API kontrolü

Performans

Otomatik Tarama - Toplu hedef işleme
Hata Toleransı - Panic recovery mekanizması
Timeout Yönetimi - Sonsuz bekleme engelleme
Goroutine Desteği - Paralel tarama potansiyeli

Veri Toplama

HTML Kaydetme - Tam sayfa kaynağı
Screenshot - Headless browser ile ekran görüntüsü
Yapılandırılmış Raporlama - JSON + Text log
HTTP Durum Kodları - Detaylı bağlantı bilgisi

Kullanıcı Dostu

Renkli Terminal Çıktısı - Görsel feedback
İlerleme Takibi - [1/10] formatında gösterim
İstatistik Özeti - Başarı/başarısızlık analizi
Esnek Input - YAML ve düz metin desteği


# Kurulum kontrolü
   go version
   
   # Kurulum (Debian/Ubuntu)
   sudo apt update
   sudo apt install golang-go -y




   # Kurulum
   sudo apt install tor -y
   
   # Başlatma
   sudo systemctl start tor
   sudo systemctl enable tor
   
   # Kontrol
   sudo systemctl status tor


   # Kali Linux
   sudo apt install chromium -y
   
   # Ubuntu
   sudo apt install chromium-browser -y
   
   # macOS
   brew install chromium




# 1. Go modülünü başlat
go mod init thor_scraper

# 2. Bağımlılıkları yükle
go get github.com/chromedp/chromedp
go get golang.org/x/net/proxy

# 3. Derleme (opsiyonel)
go build -o thor_scraper main.go



Temel Kullanım

# 1. Hedef listesi oluştur (targets.yaml)
# Taramayı başlat:
# Doğrudan çalıştırma
   go run main.go
   
   # Veya derlenmiş binary ile
   ./thor_scraper




Çıktılarınız tor_verisi klasörü adı altında tarama yapmış olduğunuz url lerin adı ile oluşturulmaktadır.

