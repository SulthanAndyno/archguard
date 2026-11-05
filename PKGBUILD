# Maintainer: Sulthan Andyno <sulthandyno@gmail.com>
pkgname=archguard-bin
pkgver=0.1.0
pkgrel=1
pkgdesc="Interactive TUI companion for Arch Linux system health, news advisories, and pacman maintenance"
arch=('x86_64')
url="https://github.com/SulthanAndyno/archguard"
license=('MIT')
depends=('glibc')
optdepends=(
    'pacman-contrib: for checkupdates and paccache support'
    'yay: for AUR updates integration'
    'paru: alternative AUR helper integration'
)
provides=('archguard')
conflicts=('archguard')
source=("$pkgname-$pkgver.tar.gz::$url/archive/refs/tags/v$pkgver.tar.gz")
sha256sums=('SKIP')

build() {
    cd "archguard-$pkgver"
    export CGO_ENABLED=0
    go build -ldflags="-s -w -X main.Version=$pkgver" -o archguard ./cmd/archguard
}

package() {
    cd "archguard-$pkgver"
    install -Dm755 archguard "$pkgdir/usr/bin/archguard"
    install -Dm644 README.md "$pkgdir/usr/share/doc/$pkgname/README.md"
}
