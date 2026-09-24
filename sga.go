package main

import (
	"fmt"
	"net/http"
	"net/mail"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
)

type Room struct {
	ID         string   `json:"id"`
	Nome       string   `json:"nome"`
	Capacidade int      `json:"capacidade"`
	Recursos   []string `json:"recursos"`
}

type Student struct {
	ID    string `json:"id"`
	Nome  string `json:"nome"`
	Email string `json:"email"`
}

type Allocation struct {
	SalaID    string `json:"sala_id"`
	DiaSemana string `json:"dia_semana"`
	Inicio    string `json:"inicio"`
	Fim       string `json:"fim"`
}

type Class struct {
	ID         string          `json:"id"`
	Nome       string          `json:"nome"`
	Disciplina string          `json:"disciplina"`
	Professor  string          `json:"professor"`
	Alunos     map[string]bool `json:"-"`
	Alocacao   *Allocation     `json:"alocacao,omitempty"`
}

type ClassView struct {
	ID               string      `json:"id"`
	Nome             string      `json:"nome"`
	Disciplina       string      `json:"disciplina"`
	Professor        string      `json:"professor"`
	QuantidadeAlunos int         `json:"quantidade_alunos"`
	Alocada          bool        `json:"alocada"`
	Alocacao         *Allocation `json:"alocacao,omitempty"`
}

type ScheduleEntry struct {
	TurmaID   string `json:"turma_id"`
	TurmaNome string `json:"turma_nome"`
	DiaSemana string `json:"dia_semana"`
	Inicio    string `json:"inicio"`
	Fim       string `json:"fim"`
}

type Store struct {
	mu       sync.RWMutex
	rooms    map[string]Room
	students map[string]Student
	classes  map[string]*Class
}

func NewStore() *Store {
	return &Store{rooms: make(map[string]Room), students: make(map[string]Student), classes: make(map[string]*Class)}
}

func fail(c *gin.Context, status int, message string) { c.JSON(status, gin.H{"erro": message}) }

func bind(c *gin.Context, value any) bool {
	if err := c.ShouldBindJSON(value); err != nil {
		fail(c, http.StatusBadRequest, "JSON inválido: "+err.Error())
		return false
	}
	return true
}

func (s *Store) createRoom(c *gin.Context) {
	var room Room
	if !bind(c, &room) {
		return
	}
	room.ID, room.Nome = strings.TrimSpace(room.ID), strings.TrimSpace(room.Nome)
	if room.ID == "" || room.Nome == "" || room.Capacidade <= 0 {
		fail(c, http.StatusBadRequest, "id, nome e capacidade maior que zero são obrigatórios")
		return
	}
	if room.Recursos == nil {
		room.Recursos = []string{}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.rooms[room.ID]; exists {
		fail(c, http.StatusConflict, "sala já cadastrada")
		return
	}
	s.rooms[room.ID] = room
	c.JSON(http.StatusCreated, room)
}

func (s *Store) listRooms(c *gin.Context) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rooms := make([]Room, 0, len(s.rooms))
	for _, room := range s.rooms {
		rooms = append(rooms, room)
	}
	sort.Slice(rooms, func(i, j int) bool { return rooms[i].ID < rooms[j].ID })
	c.JSON(http.StatusOK, rooms)
}

func (s *Store) createStudent(c *gin.Context) {
	var student Student
	if !bind(c, &student) {
		return
	}
	student.ID, student.Nome, student.Email = strings.TrimSpace(student.ID), strings.TrimSpace(student.Nome), strings.TrimSpace(student.Email)
	address, err := mail.ParseAddress(student.Email)
	if student.ID == "" || student.Nome == "" || err != nil || address.Address != student.Email {
		fail(c, http.StatusBadRequest, "id, nome e email válido são obrigatórios")
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.students[student.ID]; exists {
		fail(c, http.StatusConflict, "aluno já cadastrado")
		return
	}
	s.students[student.ID] = student
	c.JSON(http.StatusCreated, student)
}

func (s *Store) listStudents(c *gin.Context) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	students := make([]Student, 0, len(s.students))
	for _, student := range s.students {
		students = append(students, student)
	}
	sort.Slice(students, func(i, j int) bool { return students[i].ID < students[j].ID })
	c.JSON(http.StatusOK, students)
}

func (s *Store) getStudent(c *gin.Context) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	student, exists := s.students[c.Param("id")]
	if !exists {
		fail(c, http.StatusNotFound, "aluno não encontrado")
		return
	}
	c.JSON(http.StatusOK, student)
}

func (s *Store) createClass(c *gin.Context) {
	var class Class
	if !bind(c, &class) {
		return
	}
	class.ID, class.Nome = strings.TrimSpace(class.ID), strings.TrimSpace(class.Nome)
	class.Disciplina, class.Professor = strings.TrimSpace(class.Disciplina), strings.TrimSpace(class.Professor)
	if class.ID == "" || class.Nome == "" || class.Disciplina == "" || class.Professor == "" {
		fail(c, http.StatusBadRequest, "id, nome, disciplina e professor são obrigatórios")
		return
	}
	class.Alunos = make(map[string]bool)
	class.Alocacao = nil
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.classes[class.ID]; exists {
		fail(c, http.StatusConflict, "turma já cadastrada")
		return
	}
	s.classes[class.ID] = &class
	c.JSON(http.StatusCreated, viewClass(&class))
}

func viewClass(class *Class) ClassView {
	return ClassView{ID: class.ID, Nome: class.Nome, Disciplina: class.Disciplina, Professor: class.Professor,
		QuantidadeAlunos: len(class.Alunos), Alocada: class.Alocacao != nil, Alocacao: class.Alocacao}
}

func (s *Store) listClasses(c *gin.Context) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	classes := make([]ClassView, 0, len(s.classes))
	for _, class := range s.classes {
		classes = append(classes, viewClass(class))
	}
	sort.Slice(classes, func(i, j int) bool { return classes[i].ID < classes[j].ID })
	c.JSON(http.StatusOK, classes)
}

func overlap(a, b *Allocation) bool {
	return a.DiaSemana == b.DiaSemana && a.Inicio < b.Fim && a.Fim > b.Inicio
}

func studentHasConflict(classes map[string]*Class, studentID, excludedID string, requested *Allocation) bool {
	for _, other := range classes {
		if other.ID != excludedID && other.Alunos[studentID] && other.Alocacao != nil && overlap(requested, other.Alocacao) {
			return true
		}
	}
	return false
}

func (s *Store) enrollStudent(c *gin.Context) {
	var request struct {
		AlunoID string `json:"aluno_id"`
	}
	if !bind(c, &request) {
		return
	}
	request.AlunoID = strings.TrimSpace(request.AlunoID)
	if request.AlunoID == "" {
		fail(c, http.StatusBadRequest, "aluno_id é obrigatório")
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	class, classExists := s.classes[c.Param("id")]
	student, studentExists := s.students[request.AlunoID]
	if !classExists || !studentExists {
		fail(c, http.StatusNotFound, "turma ou aluno não encontrado")
		return
	}
	if class.Alunos[student.ID] {
		fail(c, http.StatusConflict, "aluno já matriculado nesta turma")
		return
	}
	if class.Alocacao != nil {
		if len(class.Alunos) >= s.rooms[class.Alocacao.SalaID].Capacidade {
			fail(c, http.StatusUnprocessableEntity, "capacidade da sala insuficiente")
			return
		}
		if studentHasConflict(s.classes, student.ID, class.ID, class.Alocacao) {
			fail(c, http.StatusConflict, "conflito de agenda do aluno")
			return
		}
	}
	class.Alunos[student.ID] = true
	c.JSON(http.StatusCreated, gin.H{"turma_id": class.ID, "aluno": student})
}

func (s *Store) classStudents(c *gin.Context) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	class, exists := s.classes[c.Param("id")]
	if !exists {
		fail(c, http.StatusNotFound, "turma não encontrada")
		return
	}
	students := make([]Student, 0, len(class.Alunos))
	for id := range class.Alunos {
		students = append(students, s.students[id])
	}
	sort.Slice(students, func(i, j int) bool { return students[i].ID < students[j].ID })
	c.JSON(http.StatusOK, students)
}

var days = map[string]string{
	"1": "segunda-feira", "segunda": "segunda-feira", "segunda-feira": "segunda-feira",
	"2": "terça-feira", "terça": "terça-feira", "terca": "terça-feira", "terça-feira": "terça-feira", "terca-feira": "terça-feira",
	"3": "quarta-feira", "quarta": "quarta-feira", "quarta-feira": "quarta-feira",
	"4": "quinta-feira", "quinta": "quinta-feira", "quinta-feira": "quinta-feira",
	"5": "sexta-feira", "sexta": "sexta-feira", "sexta-feira": "sexta-feira",
	"6": "sábado", "sábado": "sábado", "sabado": "sábado",
	"7": "domingo", "domingo": "domingo",
}

func normalizeDay(value string) (string, bool) {
	day, ok := days[strings.ToLower(strings.TrimSpace(value))]
	return day, ok
}

func validClock(value string) bool {
	if len(value) != 5 || value[2] != ':' {
		return false
	}
	for _, index := range []int{0, 1, 3, 4} {
		if value[index] < '0' || value[index] > '9' {
			return false
		}
	}
	hour, _ := strconv.Atoi(value[:2])
	minute, _ := strconv.Atoi(value[3:])
	return hour < 24 && minute < 60
}

func (s *Store) allocateClass(c *gin.Context) {
	var request struct {
		SalaID    string `json:"sala_id"`
		DiaSemana string `json:"dia_semana"`
		Inicio    string `json:"inicio"`
		Fim       string `json:"fim"`
	}
	if !bind(c, &request) {
		return
	}
	request.SalaID = strings.TrimSpace(request.SalaID)
	day, validDay := normalizeDay(request.DiaSemana)
	if request.SalaID == "" || !validDay || !validClock(request.Inicio) || !validClock(request.Fim) || request.Inicio >= request.Fim {
		fail(c, http.StatusBadRequest, "sala_id, dia_semana válido e período HH:MM crescente são obrigatórios")
		return
	}
	requested := &Allocation{SalaID: request.SalaID, DiaSemana: day, Inicio: request.Inicio, Fim: request.Fim}
	s.mu.Lock()
	defer s.mu.Unlock()
	class, classExists := s.classes[c.Param("id")]
	room, roomExists := s.rooms[request.SalaID]
	if !classExists || !roomExists {
		fail(c, http.StatusNotFound, "turma ou sala não encontrada")
		return
	}
	if len(class.Alunos) > room.Capacidade {
		fail(c, http.StatusUnprocessableEntity, "capacidade da sala insuficiente")
		return
	}
	for _, other := range s.classes {
		if other.ID == class.ID || other.Alocacao == nil || !overlap(requested, other.Alocacao) {
			continue
		}
		if other.Alocacao.SalaID == room.ID {
			fail(c, http.StatusConflict, "conflito de agenda da sala")
			return
		}
	}
	for studentID := range class.Alunos {
		if studentHasConflict(s.classes, studentID, class.ID, requested) {
			fail(c, http.StatusConflict, "conflito de agenda do aluno")
			return
		}
	}
	class.Alocacao = requested
	c.JSON(http.StatusOK, viewClass(class))
}

func (s *Store) roomSchedule(c *gin.Context) {
	var day string
	if raw := c.Query("dia_semana"); raw != "" {
		var valid bool
		day, valid = normalizeDay(raw)
		if !valid {
			fail(c, http.StatusBadRequest, "dia_semana inválido")
			return
		}
	}
	start, end := c.Query("inicio"), c.Query("fim")
	if (start == "") != (end == "") || (start != "" && (day == "" || !validClock(start) || !validClock(end) || start >= end)) {
		fail(c, http.StatusBadRequest, "informe dia_semana, inicio e fim válidos para consultar disponibilidade")
		return
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if _, exists := s.rooms[c.Param("id")]; !exists {
		fail(c, http.StatusNotFound, "sala não encontrada")
		return
	}
	entries := []ScheduleEntry{}
	available := true
	for _, class := range s.classes {
		a := class.Alocacao
		if a == nil || a.SalaID != c.Param("id") || (day != "" && a.DiaSemana != day) {
			continue
		}
		entries = append(entries, ScheduleEntry{TurmaID: class.ID, TurmaNome: class.Nome, DiaSemana: a.DiaSemana, Inicio: a.Inicio, Fim: a.Fim})
		if start != "" && overlap(&Allocation{DiaSemana: day, Inicio: start, Fim: end}, a) {
			available = false
		}
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].DiaSemana != entries[j].DiaSemana {
			return entries[i].DiaSemana < entries[j].DiaSemana
		}
		if entries[i].Inicio != entries[j].Inicio {
			return entries[i].Inicio < entries[j].Inicio
		}
		return entries[i].TurmaID < entries[j].TurmaID
	})
	response := gin.H{"sala_id": c.Param("id"), "grade": entries}
	if start != "" {
		response["disponivel"] = available
		response["periodo"] = fmt.Sprintf("%s %s-%s", day, start, end)
	}
	c.JSON(http.StatusOK, response)
}
