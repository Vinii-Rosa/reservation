# reservation
API backend para operação de restaurantes e bares. Cada estabelecimento (empresa) gerencia mesas, reservas, fila de espera, cardápio, pedidos na mesa e pagamentos.

O staff usa rotas autenticadas (/admin) com papéis dev, admin e caixa. O cliente usa rotas públicas (/public): consultar agenda e disponibilidade, fazer reserva, entrar na fila e pedir pela mesa via token/QR Code.

Inclui configuração por empresa (tolerância de atraso, tempo de retenção da mesa, mesas maiores que o grupo), auditoria de eventos, limpeza automática de reservas expiradas e notificação por e-mail (WhatsApp ainda não implementado).

Stack: Go, Gin, GORM, PostgreSQL, JWT, Docker.
