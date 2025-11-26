package router

func init() {
	factualKeywords = append(factualKeywords,
		// Questions (Who, What, Where, When)
		"quando", "onde", "quem", "qual", "quantos", "quantas", "qual é", "quem é", "onde fica",
		"quando foi", "quando aconteceu", "quando ocorreu", "quando surgiu", "quando nasceu",
		"quando morreu", "quando criou", "quando inventou", "quando descobriu", "em que ano",
		"em que data", "em que época", "em que período", "aonde", "de onde", "para onde",
		"por onde", "com quem", "de quem", "para quem", "por quem", "o qual", "a qual",
		"os quais", "as quais", "cujo", "cuja", "cujos", "cujas", "quanto", "quanta",

		// Facts & Data
		"data", "ano", "mês", "dia", "local", "lugar", "localização", "país", "cidade", "estado",
		"região", "continente", "capital", "população", "habitantes", "área", "território", "fronteira",
		"limite", "coordenadas", "latitude", "longitude", "altitude", "clima", "idioma", "moeda",
		"religião", "cultura", "gentílico", "naturalidade", "nacionalidade", "etnia", "raça",
		"cor", "gênero", "sexo", "idade", "altura", "peso", "medidas", "tamanho", "dimensões",
		"volume", "capacidade", "velocidade", "distância", "tempo", "duração", "prazo",
		"validade", "vencimento", "garantia", "fabricação", "produção", "origem", "destino",

		// Definitions & Quick Info
		"significado", "definição", "o que é", "o que significa", "conceito", "noção", "ideia básica",
		"resumo", "resumidamente", "em poucas palavras", "de forma resumida", "brevemente",
		"sucintamente", "de maneira concisa", "sinopse", "sumário", "índice", "glossário",
		"vocabulário", "dicionário", "enciclopédia", "wikipédia", "wiki", "verbete", "termo",
		"expressão", "gíria", "ditado", "provérbio", "frase", "citação", "autor", "fonte",
		"referência", "bibliografia", "créditos", "autoria", "copyright", "direitos autorais",

		// Specific Information
		"informação", "dados", "estatísticas", "números", "valores", "características", "atributos",
		"propriedades", "especificações", "detalhes básicos", "informações básicas", "dados básicos",
		"fatos", "curiosidades", "informações gerais", "ficha técnica", "perfil", "biografia",
		"currículo", "histórico", "registro", "documento", "certidão", "diploma", "certificado",
		"comprovante", "recibo", "nota fiscal", "contrato", "acordo", "tratado", "lei",
		"norma", "regra", "regulamento", "estatuto", "código", "manual", "guia", "tutorial",

		// Calendar & Time
		"janeiro", "fevereiro", "março", "abril", "maio", "junho", "julho", "agosto", "setembro", "outubro", "novembro", "dezembro",
		"segunda-feira", "terça-feira", "quarta-feira", "quinta-feira", "sexta-feira", "sábado", "domingo",
		"natal", "páscoa", "carnaval", "ano novo", "réveillon", "dia das mães", "dia dos pais", "dia dos namorados",
		"feriado nacional", "ponto facultativo", "férias", "recesso", "fim de ano", "início de ano",
		"primavera", "verão", "outono", "inverno", "estação do ano", "solstício", "equinócio",
	)
}
