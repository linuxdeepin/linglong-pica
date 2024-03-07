/*
 * SPDX-FileCopyrightText: 2022 UnionTech Software Technology Co., Ltd.
 *
 * SPDX-License-Identifier: LGPL-3.0-or-later
 */

package fs

import (
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"testing"

	"pkg.deepin.com/linglong/pica/tools/log"
)

// test IsDir
var testDataIsDir = []struct {
	in  string
	ret bool
}{
	{"/etc/default/grub.d", true},
	{"/bin/bash.txt", false},
	{"/etc/fstab", false},
	{"/usr/bin/", true},
}

func TestIsDir(t *testing.T) {
	t.Parallel()
	for _, tds := range testDataIsDir {
		fmt.Println(tds)
		ret := IsDir(tds.in)
		if ret != tds.ret {
			t.Error("failed:", tds.in)
		}
	}
}

// CheckFileExits
var testDataCheckFileExits = []struct {
	in  string
	out bool
}{
	{"/bin/bash.txt", false},
	{"/etc/fstab", true},
	{"/etc/systemd/system", true},
	{"/tmp/ll-pica", false},
}

func TestCheckFileExits(t *testing.T) {
	// t.Parallel()
	for _, tds := range testDataCheckFileExits {
		if ret, err := CheckFileExits(tds.in); err != nil && tds.out || ret != tds.out {
			t.Errorf("Failed test for CheckFileExits!")
		}
	}

}

// CreateDir
var testDataCreateDir = []struct {
	in  string
	out bool
}{
	{"/tmp/ll-pica", true},
	{"/tmp/ll-lingong", true},
	{"/etc/apt/sources.list", false},
}

func TestCreateDir(t *testing.T) {
	t.Parallel()
	for _, tds := range testDataCreateDir {
		if ret, err := CreateDir(tds.in); err != nil && tds.out || ret != tds.out {
			t.Errorf("Failed test for CreateDir! Error: %+v", tds.in)
		} else if ret {
			if ret, err := RemovePath(tds.in); !ret && err != nil {
				t.Errorf("Failed test for CreateDir! Error: failed to remove %+v", tds.in)
			}
		}
	}

}

// RemovePath
func TestRemovePath(t *testing.T) {
	t.Parallel()
	// 目录测试
	testDirPath := "/tmp/ll-pica"
	if ret, err := CreateDir(testDirPath); err != nil && !ret {
		t.Errorf("Failed test for RemovePath! Error: create dir err of %+v", testDirPath)
	}
	if ret, err := RemovePath(testDirPath); !ret && err != nil {
		t.Errorf("Failed test for RemovePath! Error: failed to remove %+v", testDirPath)
	}
	// 测试文件
	testFilePath := "/tmp/ll-pica.txt"
	if err := ioutil.WriteFile(testFilePath, []byte("I am testing!"), 0644); err != nil {
		t.Errorf("Failed test for RemovePath! Error: failed to write file of  %+v", testFilePath)
	}
	if ret, err := RemovePath(testFilePath); !ret && err != nil {
		t.Errorf("Failed test for RemovePath! Error: failed to remove %+v", testFilePath)
	}
	// 测试不存在文件
	testFilePath = "/tmp/ll-linglong.txt"
	if ret, err := RemovePath(testFilePath); ret || err == nil {
		t.Errorf("Failed test for RemovePath! Error: failed to remove %+v", testFilePath)
	}
}

// GetFileName
var testDataGetFileName = []struct {
	in  string
	out string
}{
	{"/bin/bash.txt", "bash.txt"},
	{"/etc/fstab", "fstab"},
	{"/etc/systemd/system", "system"},
	{"/usr/lib/libc.so.1.1", "libc.so.1.1"},
}

func TestGetFileName(t *testing.T) {
	t.Parallel()
	for _, tds := range testDataGetFileName {
		ret := GetFileName(tds.in)
		if ret != tds.out {
			t.Errorf("the key %v , ret %v", tds, ret)
		}
	}
}

// GetFilePPath
var testDataGetFilePPath = []struct {
	in  string
	out string
}{
	{"/bin/bash.txt", "/bin"},
	{"/etc/fstab", "/etc"},
	{"/etc/systemd/system", "/etc/systemd"},
	{"/usr/lib/libc.so.1.1", "/usr/lib"},
}

func TestGetFilePPath(t *testing.T) {
	t.Parallel()
	for _, tds := range testDataGetFilePPath {
		ret := GetFilePPath(tds.in)
		if ret != tds.out {
			t.Errorf("the key %v , ret %v", tds, ret)
		}
	}
}

// test MoveFileOrDir
var testDataMoveFileOrDir = []struct {
	src string
	dst string
	ret bool
}{
	{"/tmp/test-pica", "/tmp/test-pica1", true},
	{"/bin/bash.txt", "/tmp/bash.txt", false},
	{"/tmp/test-pica1/test-pica1.txt", "/tmp/test-pica/test-pica.txt", true},
}

func TestMoveFileOrDir(t *testing.T) {
	// t.Parallel()
	// 测试目录移动
	if ret, err := CreateDir(testDataMoveFileOrDir[0].src); !ret && err != nil {
		t.Errorf("CreateDir failed! : %s", testDataMoveFileOrDir[0].src)
	}
	ret, err := MoveFileOrDir(testDataMoveFileOrDir[0].src, testDataMoveFileOrDir[0].dst)
	if ret != testDataMoveFileOrDir[0].ret && err != nil {
		t.Error("Test move dir failed! : ", testDataMoveFileOrDir[0].src)
	}

	// 测试移动文件
	f, err := os.Create(testDataMoveFileOrDir[2].src)
	if err != nil {
		t.Error("Create file failed! : ", testDataMoveFileOrDir[2].src)
	}
	defer f.Close()
	f.WriteString("test-pica")
	ret, err = MoveFileOrDir(testDataMoveFileOrDir[2].src, testDataMoveFileOrDir[2].dst)
	if ret != testDataMoveFileOrDir[2].ret && err != nil {
		t.Error("Test move file failed! : ", testDataMoveFileOrDir[2].src)
	}

	// 移动不存在文件
	ret, err = MoveFileOrDir(testDataMoveFileOrDir[1].src, testDataMoveFileOrDir[1].dst)
	if ret != testDataMoveFileOrDir[1].ret || err == nil {
		t.Error("Test move file failed! : ", testDataMoveFileOrDir[1].src)
	}

	// 移除创建的目录
	err = os.RemoveAll(testDataMoveFileOrDir[0].src)
	if err != nil {
		t.Error("remove dir failed ! : ", testDataMoveFileOrDir[0].src)
	}
	err = os.RemoveAll(testDataMoveFileOrDir[0].dst)
	if err != nil {
		t.Error("remove dir failed ! : ", testDataMoveFileOrDir[0].dst)
	}
}

// CopyFile
var testDataCopyFile = []struct {
	in  string
	out string
	ret bool
}{
	{"/tmp/ll-pica.txt", "/tmp/ll-pica1.txt", true},
	{"/tmp/ll-lingong.txt", "/tmp/ll-lingong1.txt", false},
}

func TestCopyFile(t *testing.T) {
	// t.Parallel()
	// 测试已存在文件拷贝
	if err := ioutil.WriteFile(testDataCopyFile[0].in, []byte("ll-pica testing"), 0644); err != nil {
		t.Errorf("Failed test for TestCopyFile! Error: failed to write %+v", testDataCopyFile[0].in)
	}
	if ret, err := CopyFile(testDataCopyFile[0].in, testDataCopyFile[0].out); err != nil || !ret || ret != testDataCopyFile[0].ret {
		t.Errorf("Failed test for TestCopyFile! Error: failed to CopyFile %+v", testDataCopyFile[0].in)
	}
	// 判断文件权限
	srcFile, err := os.Open(testDataCopyFile[0].in)
	if err != nil {
		t.Errorf("Failed test for TestCopyFile! Error: failed to open %+v", testDataCopyFile[0].in)
	}
	defer srcFile.Close()
	fi1, _ := srcFile.Stat()
	perm1 := fi1.Mode()

	dstFile, err := os.Open(testDataCopyFile[0].out)
	if err != nil {
		t.Errorf("Failed test for TestCopyFile! Error: failed to open %+v", testDataCopyFile[0].out)
	}
	defer dstFile.Close()
	fi2, _ := dstFile.Stat()
	perm2 := fi2.Mode()
	if perm1 != perm2 {
		t.Errorf("Failed test for TestCopyFile! Error: failed to copy perm %+v", testDataCopyFile[0].out)
	}

	// 移除产生的文件
	if ret, err := RemovePath(testDataCopyFile[0].in); !ret || err != nil {
		t.Errorf("Failed test for TestCopyFile! Error: failed to remove %+v", testDataCopyFile[0].in)
	}
	if ret, err := RemovePath(testDataCopyFile[0].out); !ret || err != nil {
		t.Errorf("Failed test for TestCopyFile! Error: failed to remove %+v", testDataCopyFile[0].out)
	}

	// 测试不存在的文件
	if ret, err := CopyFile(testDataCopyFile[1].in, testDataCopyFile[1].out); err == nil || ret || ret != testDataCopyFile[1].ret {
		t.Errorf("Failed test for TestCopyFile! Error: failed to CopyFile %+v", testDataCopyFile[1].in)
	}
}

// CopyDir
var testDataCopyDir = []struct {
	in  string
	out string
	ret bool
}{
	{"/tmp/ll-pica", "/tmp/ll-pica1", true},
	{"/tmp/ll-linglong-test", "/tmp/ll-linglong-test1", false},
}

func TestCopyDir(t *testing.T) {
	// t.Parallel()
	// 测试已存在的目录
	if ret, err := CreateDir(testDataCopyDir[0].in); err != nil || !ret {
		t.Errorf("Failed test for TestCopyDir! Error: failed to create dir %+v", testDataCopyDir[0].in)
	}
	if ret := CopyDir(testDataCopyDir[0].in, testDataCopyDir[0].out); !ret || ret != testDataCopyDir[0].ret {
		t.Errorf("Failed test for TestCopyDir! Error: failed to copy dir %+v", testDataCopyDir[0].in)
	}
	// 移除产生的目录
	if ret, err := RemovePath(testDataCopyDir[0].in); !ret || err != nil {
		t.Errorf("Failed test for TestCopyDir! Error: failed to remove %+v", testDataCopyDir[0].in)
	}
	if ret, err := RemovePath(testDataCopyDir[0].out); !ret || err != nil {
		t.Errorf("Failed test for TestCopyDir! Error: failed to remove %+v", testDataCopyDir[0].out)
	}

	//测试不存在的目录
	if ret := CopyDir(testDataCopyDir[1].in, testDataCopyDir[1].out); ret || ret != testDataCopyDir[1].ret {
		t.Errorf("Failed test for TestCopyDir! Error: failed to copy dir %+v", testDataCopyDir[1].in)
	}
}

// test CopyFileKeepPermission
var testDataCopyFileKeepPermission = []struct {
	in  string
	ret bool
}{
	{"/bin/bash.txt", false},
	{"/etc/fstab", true},
	{"/etc/systemd/system", false},
	{"/usr/lib/x86_64-linux-gnu/libc.so.6", true},
}

func TestCopyFileKeepPermission(t *testing.T) {
	dst := "/tmp/aaaaaxxx"

	prefix := "/mnt/workdir/rootfs"
	if ret, err := CheckFileExits(prefix); err == nil && ret {
		for _, tds := range testDataCopyFileKeepPermission {
			fmt.Println(tds)
			if err := CopyFileKeepPermission(prefix+tds.in, dst, true, false); err != nil && tds.ret {
				t.Error("failed:", err, tds, dst)
			}
		}
	} else {
		log.Logger.Infof("skip test case: %v", t)
	}
}

// CopyDirKeepPathAndPerm
var testDataCopyDirKeepPathAndPerm = []struct {
	in  string
	ret bool
}{
	{"/etc/default/grub.d", true},
	{"/bin/bash.txt", false},
	{"/etc/fstab", false},
}

func TestCopyDirKeepPathAndPerm(t *testing.T) {
	t.Parallel()
	dst := "/tmp/aaaaaxxx"
	for _, tds := range testDataCopyDirKeepPathAndPerm {
		fmt.Println(tds)
		if err := CopyDirKeepPathAndPerm(tds.in, dst, true, false, false); err != nil && tds.ret {
			t.Error("failed:", err, tds, dst)
		}
	}
	if err := os.RemoveAll(dst); err != nil {
		t.Error("failed:", err, dst)
	}

}

func TestCopyDirKeepPathAndPerm2(t *testing.T) {
	// t.Parallel()
	dst := "/tmp/aaaaaxxx"
	prefix := "/mnt/workdir/rootfs"
	if ret, err := CheckFileExits(prefix); err == nil && ret {
		for _, tds := range testDataCopyDirKeepPathAndPerm {
			fmt.Println(tds)
			if err := CopyDirKeepPathAndPerm(prefix+tds.in, dst, true, false, false); err != nil && tds.ret {
				t.Error("failed:", err, tds, dst)
			}
		}
		if err := os.RemoveAll(dst); err != nil {
			t.Error("failed:", err, dst)
		}
	} else {
		log.Logger.Infof("skip test case: %v", t)
	}

}

// FindBundlePath
func TestFindBundlePath(t *testing.T) {
	findBundleTestDir, err := ioutil.TempDir("/tmp/", "ll-pica_")
	if err != nil {
		t.Errorf("failed to create temporary file: %v", err)
	}

	t.Parallel()
	// 测试未存在uab目录
	dirPath := findBundleTestDir
	if ret, err := CreateDir(dirPath); !ret || err != nil {
		t.Errorf("Failed test for TestFindBundlePath! Error: failed to create dir  %+v", dirPath)
	}
	defer func() { RemovePath(dirPath) }()

	if ret, err := CreateDir(dirPath + "/linglong"); !ret || err != nil {
		t.Errorf("Failed test for TestFindBundlePath! Error: failed to create dir  %+v", dirPath)
	}
	uabList, err := FindBundlePath(dirPath)
	if err == nil || len(uabList) != 0 {
		t.Errorf("Failed test for TestFindBundlePath! Error: failed to FindBundlePath  /tmp/ll-pica")
	}
	// 测试已存在uab目录
	// 创建uab文件
	uab1File := dirPath + "/ll-pica_1.2.1_amd64.uab"
	uab2File := dirPath + "/linglong/ll-pica_1.1.1_amd64.uab"
	if err := ioutil.WriteFile(uab1File, []byte("I am uab1"), 0755); err != nil {
		t.Errorf("Failed test for TestFindBundlePath! Error: failed to create file  %+v", uab1File)
	}
	if err := ioutil.WriteFile(uab2File, []byte("I am uab2"), 0755); err != nil {
		t.Errorf("Failed test for TestFindBundlePath! Error: failed to create file  %+v", uab2File)
	}
	// 搜索uab
	uabList, err = FindBundlePath(dirPath)
	if err != nil {
		t.Errorf("Failed test for TestFindBundlePath! Error: failed to FindBundlePath  /tmp/ll-pica")
	}
	if uabList[1] != uab1File {
		t.Errorf("Failed test for TestFindBundlePath! Error: failed to FindBundlePath  %+v", uab1File)
	}
	if uabList[0] != uab2File {
		t.Errorf("Failed test for TestFindBundlePath! Error: failed to FindBundlePath  %+v", uab2File)
	}
	// // 移除目录
	// if ret, err := RemovePath(dirPath); !ret || err != nil {
	// 	t.Errorf("Failed test for TestFindBundlePath! Error: failed to remove dir /tmp/ll-pica")
	// }

}

// HasBundleName
var testDataHasBundleName = []struct {
	in  string
	ret bool
}{
	{"/usr/share/ll-pica.uab", true},
	{"/tmp/test/ll-pica.uac", false},
	{"/etc/fstab", false},
}

func TestHasBundleName(t *testing.T) {
	t.Parallel()
	for _, tds := range testDataHasBundleName {
		if ret := HasBundleName(tds.in); ret != tds.ret {
			t.Errorf("Failed test for TestHasBundleName! Error: failed to HasBundleName %+v", tds.in)
		}
	}
}

// DesktopInit
var testDataDesktopInit = []struct {
	Group string
	key   string
	value string
}{
	{"Desktop Entry", "Exec", "/usr/bin/google-chrome-stable %U"},
	{"Desktop Entry", "Name", "Google Chrome"},
	{"Desktop Action new-window", "Exec", "/usr/bin/google-chrome-stable"},
}

const TMPL_DESKTOP = `[Desktop Entry]
Version=1.0
Name=Google Chrome
# Only KDE 4 seems to use GenericName, so we reuse the KDE strings.
# From Ubuntu's language-pack-kde-XX-base packages, version 9.04-20090413.
GenericName=Web Browser
GenericName[ar]=متصفح الشبكة
GenericName[bg]=Уеб браузър
GenericName[ca]=Navegador web
GenericName[cs]=WWW prohlížeč
GenericName[da]=Browser
GenericName[de]=Web-Browser
GenericName[el]=Περιηγητής ιστού
GenericName[en_GB]=Web Browser
GenericName[es]=Navegador web
GenericName[et]=Veebibrauser
GenericName[fi]=WWW-selain
GenericName[fr]=Navigateur Web
GenericName[gu]=વબ બરાઉઝર
GenericName[he]=דפדפן אינטרנט
GenericName[hi]=वब बराउजर
GenericName[hu]=Webböngésző
GenericName[it]=Browser Web
GenericName[ja]=ウェブブラウザ
GenericName[kn]=ಜಾಲ ವೀಕಷಕ
GenericName[ko]=웹 브라우저
GenericName[lt]=Žiniatinklio naršyklė
GenericName[lv]=Tīmekļa pārlūks
GenericName[ml]=വെബ ബരൌസര
GenericName[mr]=वब बराऊजर
GenericName[nb]=Nettleser
GenericName[nl]=Webbrowser
GenericName[pl]=Przeglądarka WWW
GenericName[pt]=Navegador Web
GenericName[pt_BR]=Navegador da Internet
GenericName[ro]=Navigator de Internet
GenericName[ru]=Веб-браузер
GenericName[sl]=Spletni brskalnik
GenericName[sv]=Webbläsare
GenericName[ta]=இணைய உலாவி
GenericName[th]=เวบเบราวเซอร
GenericName[tr]=Web Tarayıcı
GenericName[uk]=Навігатор Тенет
GenericName[zh_CN]=网页浏览器
GenericName[zh_HK]=網頁瀏覽器
GenericName[zh_TW]=網頁瀏覽器
# Not translated in KDE, from Epiphany 2.26.1-0ubuntu1.
GenericName[bn]=ওয়েব বরাউজার
GenericName[fil]=Web Browser
GenericName[hr]=Web preglednik
GenericName[id]=Browser Web
GenericName[or]=ଓବେବ ବରାଉଜର
GenericName[sk]=WWW prehliadač
GenericName[sr]=Интернет прегледник
GenericName[te]=మహతల అనవష
GenericName[vi]=Bộ duyệt Web
# Gnome and KDE 3 uses Comment.
Comment=Access the Internet
Comment[ar]=الدخول إلى الإنترنت
Comment[bg]=Достъп до интернет
Comment[bn]=ইনটারনেটটি অযাকসেস করন
Comment[ca]=Accedeix a Internet
Comment[cs]=Přístup k internetu
Comment[da]=Få adgang til internettet
Comment[de]=Internetzugriff
Comment[el]=Πρόσβαση στο Διαδίκτυο
Comment[en_GB]=Access the Internet
Comment[es]=Accede a Internet.
Comment[et]=Pääs Internetti
Comment[fi]=Käytä internetiä
Comment[fil]=I-access ang Internet
Comment[fr]=Accéder à Internet
Comment[gu]=ઇટરનટ ઍકસસ કરો
Comment[he]=גישה אל האינטרנט
Comment[hi]=इटरनट तक पहच सथापित कर
Comment[hr]=Pristup Internetu
Comment[hu]=Internetelérés
Comment[id]=Akses Internet
Comment[it]=Accesso a Internet
Comment[ja]=インターネットにアクセス
Comment[kn]=ಇಂಟರನಟ ಅನನು ಪರವೇಶಸ
Comment[ko]=인터넷 연결
Comment[lt]=Interneto prieiga
Comment[lv]=Piekļūt internetam
Comment[ml]=ഇനറരനെററ ആകസസ ചെയയക
Comment[mr]=इटरनटमधय परवश करा
Comment[nb]=Gå til Internett
Comment[nl]=Verbinding maken met internet
Comment[or]=ଇଣଟରନେଟ ପରବେଶ କରନତ
Comment[pl]=Skorzystaj z internetu
Comment[pt]=Aceder à Internet
Comment[pt_BR]=Acessar a internet
Comment[ro]=Accesaţi Internetul
Comment[ru]=Доступ в Интернет
Comment[sk]=Prístup do siete Internet
Comment[sl]=Dostop do interneta
Comment[sr]=Приступите Интернету
Comment[sv]=Gå ut på Internet
Comment[ta]=இணையததை அணுகுதல
Comment[te]=ఇంటరనటను ఆకసస చయయండ
Comment[th]=เขาถงอนเทอรเนต
Comment[tr]=İnternet'e erişin
Comment[uk]=Доступ до Інтернету
Comment[vi]=Truy cập Internet
Comment[zh_CN]=访问互联网
Comment[zh_HK]=連線到網際網路
Comment[zh_TW]=連線到網際網路
Exec=/usr/bin/google-chrome-stable %U
StartupNotify=true
Terminal=false
Icon=google-chrome
Type=Application
Categories=Network;WebBrowser;
MimeType=text/html;text/xml;application/xhtml_xml;image/webp;x-scheme-handler/http;x-scheme-handler/https;x-scheme-handler/ftp;
Actions=new-window;new-private-window;

[Desktop Action new-window]
Name=New Window
Name[am]=አዲስ መስኮት
Name[ar]=نافذة جديدة
Name[bg]=Нов прозорец
Name[bn]=নতন উইনডো
Name[ca]=Finestra nova
Name[cs]=Nové okno
Name[da]=Nyt vindue
Name[de]=Neues Fenster
Name[el]=Νέο Παράθυρο
Name[en_GB]=New Window
Name[es]=Nueva ventana
Name[et]=Uus aken
Name[fa]=پنجره جدید
Name[fi]=Uusi ikkuna
Name[fil]=New Window
Name[fr]=Nouvelle fenêtre
Name[gu]=નવી વિડો
Name[hi]=नई विडो
Name[hr]=Novi prozor
Name[hu]=Új ablak
Name[id]=Jendela Baru
Name[it]=Nuova finestra
Name[iw]=חלון חדש
Name[ja]=新規ウインドウ
Name[kn]=ಹೊಸ ವಂಡೊ
Name[ko]=새 창
Name[lt]=Naujas langas
Name[lv]=Jauns logs
Name[ml]=പതിയ വിനഡോ
Name[mr]=नवीन विडो
Name[nl]=Nieuw venster
Name[no]=Nytt vindu
Name[pl]=Nowe okno
Name[pt]=Nova janela
Name[pt_BR]=Nova janela
Name[ro]=Fereastră nouă
Name[ru]=Новое окно
Name[sk]=Nové okno
Name[sl]=Novo okno
Name[sr]=Нови прозор
Name[sv]=Nytt fönster
Name[sw]=Dirisha Jipya
Name[ta]=புதிய சாளரம
Name[te]=కరతత వండ
Name[th]=หนาตางใหม
Name[tr]=Yeni Pencere
Name[uk]=Нове вікно
Name[vi]=Cửa sổ Mới
Name[zh_CN]=新建窗口
Name[zh_TW]=開新視窗
Exec=/usr/bin/google-chrome-stable

[Desktop Action new-private-window]
Name=New Incognito Window
Name[ar]=نافذة جديدة للتصفح المتخفي
Name[bg]=Нов прозорец „инкогнито“
Name[bn]=নতন ছদমবেশী উইনডো
Name[ca]=Finestra d'incògnit nova
Name[cs]=Nové anonymní okno
Name[da]=Nyt inkognitovindue
Name[de]=Neues Inkognito-Fenster
Name[el]=Νέο παράθυρο για ανώνυμη περιήγηση
Name[en_GB]=New Incognito window
Name[es]=Nueva ventana de incógnito
Name[et]=Uus inkognito aken
Name[fa]=پنجره جدید حالت ناشناس
Name[fi]=Uusi incognito-ikkuna
Name[fil]=Bagong Incognito window
Name[fr]=Nouvelle fenêtre de navigation privée
Name[gu]=નવી છપી વિડો
Name[hi]=नई गपत विडो
Name[hr]=Novi anoniman prozor
Name[hu]=Új Inkognitóablak
Name[id]=Jendela Penyamaran baru
Name[it]=Nuova finestra di navigazione in incognito
Name[iw]=חלון חדש לגלישה בסתר
Name[ja]=新しいシークレット ウィンドウ
Name[kn]=ಹೊಸ ಅಜಞಾತ ವಂಡೋ
Name[ko]=새 시크릿 창
Name[lt]=Naujas inkognito langas
Name[lv]=Jauns inkognito režīma logs
Name[ml]=പതിയ വേഷ പരചഛനന വിനഡോ
Name[mr]=नवीन गपत विडो
Name[nl]=Nieuw incognitovenster
Name[no]=Nytt inkognitovindu
Name[pl]=Nowe okno incognito
Name[pt]=Nova janela de navegação anónima
Name[pt_BR]=Nova janela anônima
Name[ro]=Fereastră nouă incognito
Name[ru]=Новое окно в режиме инкогнито
Name[sk]=Nové okno inkognito
Name[sl]=Novo okno brez beleženja zgodovine
Name[sr]=Нови прозор за прегледање без архивирања
Name[sv]=Nytt inkognitofönster
Name[ta]=புதிய மறைநிலைச சாளரம
Name[te]=కరతత అజఞత వండ
Name[th]=หนาตางใหมทไมระบตวตน
Name[tr]=Yeni Gizli pencere
Name[uk]=Нове вікно в режимі анонімного перегляду
Name[vi]=Cửa sổ ẩn danh mới
Name[zh_CN]=新建隐身窗口
Name[zh_TW]=新增無痕式視窗
Exec=/usr/bin/google-chrome-stable --incognito
`

func TestDesktopInit(t *testing.T) {
	// t.Parallel()
	// 创建目录
	dirPath := "/tmp/ll-pica"
	if ret, err := CreateDir(dirPath); !ret || err != nil {
		t.Errorf("Failed test for TestDesktopInit! Error: failed to create dir  %+v", dirPath)
	}
	defer func() { RemovePath(dirPath) }()
	// 新建desktop file文件
	desktopPath := "/tmp/ll-pica/ll-pica.desktop"
	if err := ioutil.WriteFile(desktopPath, []byte(TMPL_DESKTOP), 0644); err != nil {
		t.Errorf("Failed test for TestDesktopInit! Error: failed to create file  %+v", desktopPath)
	}
	// 初始化desktop
	ret, data := DesktopInit(desktopPath)
	if !ret {
		t.Errorf("Failed test for TestDesktopInit! Error: failed to DesktopInit  %+v", desktopPath)
	}
	for _, tds := range testDataDesktopInit {
		if data[tds.Group][tds.key] != tds.value {
			t.Errorf("Failed test for TestDesktopInit! Error: failed to DesktopInit  %+v", desktopPath)
		}
	}
	// 移除目录
	// if ret, err := RemovePath(dirPath); !ret || err != nil {
	// 	t.Errorf("Failed test for TestDesktopInit! Error: failed to remove dir %+v", dirPath)
	// }
}

// DesktopGroupname
func TestDesktopGroupname(t *testing.T) {
	// t.Parallel()
	// 创建目录
	dirPath := "/tmp/ll-pica"
	if ret, err := CreateDir(dirPath); !ret || err != nil {
		t.Errorf("Failed test for TestDesktopGroupname! Error: failed to create dir  %+v", dirPath)
	}
	// 新建desktop file文件
	desktopPath := "/tmp/ll-pica/ll-pica.desktop"
	if err := ioutil.WriteFile(desktopPath, []byte(TMPL_DESKTOP), 0644); err != nil {
		t.Errorf("Failed test for TestDesktopGroupname! Error: failed to create file  %+v", desktopPath)
	}
	// 获取desktop groupname
	data := DesktopGroupname(desktopPath)
	if len(data) == 0 {
		t.Errorf("Failed test for TestDesktopGroupname! Error: failed to get group name of   %+v", desktopPath)
	}
	sort.Strings(data)
	// 搜索存在group name
	index := sort.SearchStrings(data, "Desktop Action new-window")
	if data[index] != "Desktop Action new-window" {
		t.Errorf("Failed test for TestDesktopGroupname! Error: failed to get group name of   %+v", desktopPath)
	}

	// 搜索不存在group name
	index = sort.SearchStrings(data, "ljjsdf")
	if index != len(data) {
		t.Errorf("Failed test for TestDesktopGroupname! Error: failed to get group name of   %+v", desktopPath)
	}

	// 移除目录
	if ret, err := RemovePath(dirPath); !ret || err != nil {
		t.Errorf("Failed test for TestDesktopGroupname! Error: failed to remove dir %+v", dirPath)
	}
}

// TransExecToLl
var testDataTransExecToLl = []struct {
	appid string
	in    string
	out   string
}{
	{"org.deepin.calculator", "/opt/app/org.deepin.calculator/files/bin/deepin-calculator --test %u", "ll-cli run org.deepin.calculator --exec \"/opt/app/org.deepin.calculator/files/bin/deepin-calculator --test\" %u"},
	{"org.deepin.calculator", "/usr/bin/deepin-calculator", "ll-cli run org.deepin.calculator --exec \"deepin-calculator\""},
	{"org.deepin.calculator", "/usr/bin/deepin-calculator --debug %f", "ll-cli run org.deepin.calculator --exec \"deepin-calculator --debug\" %f"},
}

// TransIconToLl
var testDataTransIconToLl = []struct {
	in  string
	out string
}{
	{"/usr/share/icons/hicolor/96x96/apps/deepin-toggle-desktop.png", "deepin-toggle-desktop"},
	{"/usr/share/icons/hicolor/96x96/apps/deepin-toggle-desktop.svg", "deepin-toggle-desktop"},
	{"/opt/apps/org.deepin.calculator/files/icons/deepin-calculator.png", "/opt/apps/org.deepin.calculator/files/icons/deepin-calculator.png"},
}

func TestTransIconToLl(t *testing.T) {
	t.Parallel()
	for _, tds := range testDataTransIconToLl {
		if ret := TransIconToLl(tds.in); ret != tds.out {
			t.Errorf("Failed test for TestTransIconToLl! Error: failed to TransIconToLl %+v", tds.in)
		}
	}

}
