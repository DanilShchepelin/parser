package main

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"github.com/tebeka/selenium"
	"github.com/tebeka/selenium/chrome"
	"log"
	"os"
	"regexp"
	"time"
)

type Product struct {
	category string
	name     string
	url      string
	imageUrl string
	price    string
	oldPrice string
}

func main() {
	url := "https://samokat.ru/"

	service, err := selenium.NewChromeDriverService("./chromedriver", 4444)
	if err != nil {
		log.Fatalf("ошибка создания драйвера chrome: %v", err)
	}
	defer func(service *selenium.Service) {
		err := service.Stop()
		if err != nil {
			log.Fatalf("ошибка остановки сервиса: %v", err)
		}
	}(service)

	customUserAgent := "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36"
	caps := selenium.Capabilities{}
	caps.AddChrome(chrome.Capabilities{Args: []string{
		"--user-agent=" + customUserAgent,
	}})
	caps.AddProxy(selenium.Proxy{
		Type:     "manual",
		HTTP:     "155.94.241.130",
		HTTPPort: 3128,
	})

	driver, err := selenium.NewRemote(caps, "")
	if err != nil {
		log.Fatalf("ошибка создания удаленного клиента: %v", err)
	}

	err = driver.MaximizeWindow("")
	if err != nil {
		log.Fatalf("ошибка установки разрешения окна браузера: %v", err)
	}

	err = driver.Get(url)
	if err != nil {
		log.Fatalf("ошибка загрузки страницы': %v", err)
	}

	setAddress(driver)

	// ждем полной перезагрузки страницы
	time.Sleep(4 * time.Second)

	categoriesLinks := getCategoriesLinks(driver)

	products := parseProducts(driver, categoriesLinks)

	createCsvFile(products)
}

func getProductName(product selenium.WebElement) string {
	productName, err := product.FindElement(selenium.ByCSSSelector, ".ProductCard_name__czrVx")
	if err != nil {
		log.Fatalf("ошибка получшения названия продукта: %v", err)
	}
	name, err := productName.Text()
	if err != nil {
		log.Fatalf("ошибка преобразования текста элемента: %v", err)
	}
	return name
}

func getProductLink(product selenium.WebElement) string {
	productLink, err := product.GetAttribute("href")
	if err != nil {
		log.Fatalf("ошибка получения ссылки на продукт: %v", err)
	}
	return productLink
}

func getProductPrices(product selenium.WebElement) (string, string) {
	prices := getPricesElements(product)

	if len(prices) > 1 {
		oldPrice, err := prices[0].Text()
		if err != nil {
			log.Fatalf("ошибка получения старой цены: %v", err)
		}
		currentPrice := getCurrentPriceFromElement(prices[1])
		return currentPrice, oldPrice
	}
	currentPrice := getCurrentPriceFromElement(prices[0])
	return currentPrice, ""
}

func getPricesElements(product selenium.WebElement) []selenium.WebElement {
	price, err := product.FindElement(selenium.ByCSSSelector, ".ProductCard_actions__2AbGZ")
	if err != nil {
		log.Fatalf("ошибка получения описания товара: %v", err)
	}
	pricesSpan, err := price.FindElement(selenium.ByTagName, "span")
	if err != nil {
		log.Fatalf("ошибка получения элемента с ценами: %v", err)
	}
	prices, err := pricesSpan.FindElements(selenium.ByTagName, "span")
	if err != nil {
		log.Fatalf("ошибка получения цен: %v", err)
	}
	return prices
}

func getCurrentPriceFromElement(priceElement selenium.WebElement) string {
	currentPrice, err := priceElement.Text()
	if err != nil {
		log.Fatalf("ошибка получения цены: %v", err)
	}
	re := regexp.MustCompile(`\d`)
	numbers := re.FindAllString(currentPrice, -1)
	result := ""
	for _, num := range numbers {
		result += num
	}
	return result
}

func getProductImageUrl(product selenium.WebElement) string {
	productImageElement, err := product.FindElement(selenium.ByCSSSelector, ".ProductCardImage_root__b96bY")
	if err != nil {
		log.Fatalf("ошибка получения элемента изображения: %v", err)
	}
	productImageTag, err := productImageElement.FindElement(selenium.ByTagName, "img")
	if err != nil {
		log.Fatalf("ошибка получения изображеня: %v", err)
	}
	imageUrl, err := productImageTag.GetAttribute("src")
	if err != nil {
		log.Fatalf("ошибка получения ссылки на изображение: %v", err)
	}
	return imageUrl
}

func setAddress(driver selenium.WebDriver) {
	if err := driver.WaitWithTimeout(isSidebarDisplayed, 10*time.Second); err != nil {
		log.Fatalf("ошибка ожидания загрузки элемента: %v", err)
	}

	clickEmptyAddressPlug(driver)

	if err := driver.WaitWithTimeout(isAddressSuggestionDisplayed, 5*time.Second); err != nil {
		log.Fatalf("ошибка загрузки элемента выбора адреса: %v", err)
	}

	addressBlock, err := driver.FindElement(selenium.ByCSSSelector, ".AddressCreation_root__RdVV2")
	if err != nil {
		log.Fatalf("ошибка получения элемента указания адреса: %v", err)
	}

	setCityToInput(addressBlock)
	selectCityFromList(driver)

	time.Sleep(3 * time.Second)

	inputElement := setAddressToInput(addressBlock)
	selectAddressFromList(inputElement)

	time.Sleep(3 * time.Second)

	submitAddress(driver)
}

func isSidebarDisplayed(driver selenium.WebDriver) (bool, error) {
	element, err := driver.FindElement(selenium.ByCSSSelector, ".DesktopScreen_sidebar__vUXIl")
	if err != nil {
		return false, err
	}
	return element.IsDisplayed()
}

func clickEmptyAddressPlug(driver selenium.WebDriver) {
	emptyAddressPlug, err := driver.FindElement(selenium.ByCSSSelector, ".EmptyAddressPlug_map__IMk_l")
	if err != nil {
		log.Fatalf("ошибка получения элемента карты: %v", err)
	}
	err = emptyAddressPlug.Click()
	if err != nil {
		log.Fatalf("ошибка нажатия на карту: %v", err)
	}
}

func isAddressSuggestionDisplayed(driver selenium.WebDriver) (bool, error) {
	element, err := driver.FindElement(selenium.ByCSSSelector, ".AddressSuggest_root__9pSaE")
	if err != nil {
		return false, err
	}
	return element.IsDisplayed()
}

func setCityToInput(rootElement selenium.WebElement) {
	inputElement, err := rootElement.FindElement(selenium.ByCSSSelector, "._textInputContainer--size-m_1frhv_1")
	if err != nil {
		log.Fatalf("ошибка получения элемента указания города: %v", err)
	}

	input, err := inputElement.FindElement(selenium.ByTagName, "input")
	if err != nil {
		log.Fatalf("ошибка получения поля указания города: %v", err)
	}
	time.Sleep(1 * time.Second)
	// иногда по умолчанию ставится Москва
	err = input.SendKeys("")
	if err != nil {
		log.Fatalf("ошибка указания города: %v", err)
	}

	fmt.Println("Введите город")
	var city string
	_, _ = fmt.Scan(&city)

	err = input.SendKeys(city)
	if err != nil {
		log.Fatalf("ошибка указания города: %v", err)
	}
}

func selectCityFromList(driver selenium.WebDriver) {
	err := waitAddressSuggestionElement(driver)
	if err != nil {
		log.Fatalf("ошибка ожидания загрузки элемента указания города: %v", err)
	}

	cities, err := driver.FindElements(selenium.ByCSSSelector, ".Suggest_suggestItem__hOaW9")
	if err != nil {
		log.Fatalf("ошибка получения списка городов: %v", err)
	}

	if len(cities) == 0 {
		log.Fatal("ошибка получения списка городов")
	}

	showCitiesList(cities)

	selectCurrentCity(cities)
}

func waitAddressSuggestionElement(driver selenium.WebDriver) error {
	err := driver.WaitWithTimeout(func(driver selenium.WebDriver) (bool, error) {
		element, _ := driver.FindElement(selenium.ByCSSSelector, ".AddressSuggest_root__9pSaE")
		if element != nil {
			return element.IsDisplayed()
		}
		return false, nil
	}, 3*time.Second)
	return err
}

func showCitiesList(cities []selenium.WebElement) {
	fmt.Println("Список городов")
	for i, city := range cities {
		text, err := city.Text()
		if err != nil {
			log.Fatalf("ошибка получения текста города: %v", err)
		}
		fmt.Printf("%d: %s\n", i+1, text)
	}
}

func selectCurrentCity(cities []selenium.WebElement) {
	fmt.Println("Выберите город из списка")
	var selectedCity int
	_, _ = fmt.Scan(&selectedCity)

	if selectedCity < 1 || selectedCity > len(cities) {
		log.Fatalf("ошибка выбора нужного города: %d", selectedCity)
	}

	for index, city := range cities {
		if index+1 == selectedCity {
			if err := city.Click(); err != nil {
				log.Fatalf("ошибка выбора нужного города: %v", err)
			}
			break
		}
	}
}

func setAddressToInput(rootElement selenium.WebElement) selenium.WebElement {
	inputElements, err := rootElement.FindElements(selenium.ByCSSSelector, ".Suggest_root__KuclW")
	if err != nil {
		log.Fatalf("ошибка получения элеменотов заполнения адреса: %v", err)
	}
	input, err := inputElements[1].FindElement(selenium.ByTagName, "input")
	if err != nil {
		log.Fatalf("ошибка получения полей заполнения адреса: %v", err)
	}

	fmt.Println("Введите улицу и дом")
	myScanner := bufio.NewScanner(os.Stdin)

	myScanner.Scan()
	street := myScanner.Text()

	err = input.SendKeys(street)
	if err != nil {
		log.Fatalf("ошибка установки значения адреса: %v", err)
	}

	time.Sleep(3 * time.Second)

	return inputElements[1]
}

func selectAddressFromList(inputElement selenium.WebElement) {
	addresses, err := inputElement.FindElements(selenium.ByCSSSelector, ".Suggest_suggestItem__hOaW9")
	if err != nil {
		log.Fatalf("ошибка получения адресов: %v", err)
	}

	if len(addresses) == 0 {
		log.Fatal("ошибка получения списка адресов")
	}

	showAddressesList(addresses)

	selectCurrentAddress(addresses)
}

func showAddressesList(addresses []selenium.WebElement) {
	fmt.Println("Список похожих адресов")
	for i := 0; i < len(addresses); i++ {
		streetName, err := addresses[i].FindElements(selenium.ByTagName, "span")
		if err != nil {
			log.Fatalf("ошибка получения заголовков адресов: %v", err)
		}
		text, err := streetName[0].Text()
		if err != nil {
			log.Fatalf("ошибка получения текста заголовков адресов: %v", err)
		}
		fmt.Printf("%d: %s\n", i+1, text)
	}
}

func selectCurrentAddress(addresses []selenium.WebElement) {
	fmt.Println("Выберите точный адрес из списка")
	var selectedStreet int
	_, _ = fmt.Scan(&selectedStreet)

	if selectedStreet < 1 || selectedStreet > len(addresses) {
		log.Fatalf("ошибка выбора нужного города: %d", selectedStreet)
	}

	for index, address := range addresses {
		if index+1 == selectedStreet {
			err := address.Click()
			if err != nil {
				log.Fatalf("ошибка выбора адреса: %v", err)
			}
			break
		}
	}
}

func submitAddress(driver selenium.WebDriver) {
	addressBlock, err := driver.FindElement(selenium.ByCSSSelector, ".AddressCreation_info__tX0zZ")
	if err != nil {
		log.Fatalf("ошибка получения элемента добавления адреса: %v", err)
	}

	submitButton, err := addressBlock.FindElement(selenium.ByTagName, "button")
	if err != nil {
		log.Fatalf("ошибка получения кнопки подтверждения: %v", err)
	}

	err = submitButton.Click()
	if err != nil {
		log.Fatalf("ошибка нажатия подтверждения адреса: %v", err)
	}
}

func getCategoriesLinks(driver selenium.WebDriver) []string {
	categoriesCatalog, err := driver.FindElements(selenium.ByCSSSelector, ".CatalogTreeSectionCard_categories__4uYFm")
	if err != nil {
		log.Fatalf("ошибка получения элементов категорий: %v", err)
	}

	var categoriesLinks []string

	for i := 0; i <= 2; i++ {
		categories, err := categoriesCatalog[i].FindElements(selenium.ByTagName, "a")
		if err != nil {
			log.Fatalf("ошибка получения категории': %v", err)
		}
		for _, category := range categories {
			link, err := category.GetAttribute("href")
			if err != nil {
				log.Fatalf("ошибка получения ссылки на категорию': %v", err)
			}
			categoriesLinks = append(categoriesLinks, link)
		}
	}

	return categoriesLinks
}

func parseProducts(driver selenium.WebDriver, categoriesLinks []string) []Product {
	var products []Product
	var totalCount int

	fmt.Println("Начинаем парсинг продуктов")
	for _, link := range categoriesLinks {
		categoryTitle, productsLists := parseCategory(driver, link)

		fmt.Println("Категория: ", categoryTitle)
		productsCount := 0

		for _, productsList := range productsLists {
			productsItems := parseProductsList(productsList)
			productsCount += len(productsItems)
			totalCount += len(productsItems)

			for _, product := range productsItems {
				productItem := createProductItem(categoryTitle, product)
				products = append(products, productItem)
			}
		}
		fmt.Println("Количество продуктов в категории:", productsCount)
	}
	fmt.Println("Полученное количество продуктов:", totalCount)
	fmt.Println("Парсинг продуктов завершен")

	return products
}

func parseCategory(driver selenium.WebDriver, link string) (string, []selenium.WebElement) {
	err := driver.Get(link)
	if err != nil {
		log.Fatalf("ошибка загрузки страницы': %v", err)
	}
	time.Sleep(2 * time.Second)

	categoryTitleElem, err := driver.FindElement(selenium.ByCSSSelector, ".CategoryPage_categoryNameContainer__C35DT")
	categoryTitle, err := categoryTitleElem.Text()

	productsLists, err := driver.FindElements(selenium.ByCSSSelector, ".ProductsList_productList__XIJx_")
	if err != nil {
		log.Fatalf("ошибка получения списков продуктов': %v", err)
	}

	return categoryTitle, productsLists
}

func parseProductsList(productsList selenium.WebElement) []selenium.WebElement {
	productsItems, err := productsList.FindElements(selenium.ByTagName, "a")
	if err != nil {
		log.Fatalf("ошибка получения продуктов': %v", err)
	}
	return productsItems
}

func createProductItem(categoryTitle string, product selenium.WebElement) Product {
	name := getProductName(product)
	productLink := getProductLink(product)
	currentPrice, oldPrice := getProductPrices(product)
	imageUrl := getProductImageUrl(product)

	productItem := Product{
		category: categoryTitle,
		name:     name,
		url:      productLink,
		price:    currentPrice,
		oldPrice: oldPrice,
		imageUrl: imageUrl,
	}

	return productItem
}

func createCsvFile(products []Product) {
	file, err := os.Create("products.csv")
	if err != nil {
		log.Fatalf("ошибка создания файла: %v", err)
	}

	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			log.Fatalf("ошибка закрытия файла: %v", err)
		}
	}(file)

	writer := csv.NewWriter(file)

	headers := []string{
		"category",
		"name",
		"url",
		"price",
		"oldPrice",
		"imageUrl",
	}

	err = writer.Write(headers)
	if err != nil {
		log.Fatalf("ошибка записи заголовков в файл: %v", err)
	}

	for _, product := range products {
		record := []string{
			product.category,
			product.name,
			product.url,
			product.price,
			product.oldPrice,
			product.imageUrl,
		}

		err = writer.Write(record)
		if err != nil {
			log.Fatalf("ошибка записи в файл: %v", err)
		}
	}

	defer writer.Flush()
}
