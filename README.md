# detour2-fyne

## Build

### Windows

```bash
scoop install mingw
mingw64
go install fyne.io/fyne/v2/cmd/fyne@latest
go mod tidy
fyne package --release
```

### Android

compile under Linux(wsl2)

```bash
go install fyne.io/fyne/v2/cmd/fyne@latest
go mod tidy
export ANDROID_NDK_HOME=/PATH/TO/android-ndk-r19c/
fyne package -os android/arm64 -appID com.detour2 -icon Icon.png --release
```
