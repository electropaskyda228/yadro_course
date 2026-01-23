# Тестирование

## Цель

Покрыть код модульными тестами.

Тесты должны запускаться через make unit и проходить. Каждый файл должен быть покрыт как
минимум на 50%. Не покрываются тестами автосгенерированные файлы. Файл покрытия тестами
cover.html должен быть закоммичен в текущее решение.

Сервисы должны собираться и запускаться через модифицированный compose файл,
а также проходить интеграционные тесты - запуск специального тест контейнера.

## Критерии приемки

1. make unit запускает модульные тесты и собирает статистику.
2. Код покрыт тестами на 50%.

## Материалы для ознакомления

- [Hello world test](https://go.dev/doc/tutorial/add-a-test)
- [Comprehensive Guide to Testing in Go](https://blog.jetbrains.com/go/2022/11/22/comprehensive-guide-to-testing-in-go/)
- [Unit-Тестирование в Golang](https://www.youtube.com/watch?v=fMUNBJPhP6Y)
- [Unit-Тестирование Веб-Приложений в Golang](http://youtube.com/watch?v=Mvw5fbHGJFw)
- [Unit-Тестирование Работы с БД в Golang](http://youtube.com/watch?v=QJq3PZ1V-5Y)
