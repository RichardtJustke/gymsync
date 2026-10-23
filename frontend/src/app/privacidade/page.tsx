"use client";

import React from "react";
import Link from "next/link";
import { Shield, ArrowLeft, Lock, FileText, CheckCircle2, UserX, Database } from "lucide-react";

export default function PrivacyPage() {
  return (
    <div className="min-h-screen bg-ink-950 text-ink-100 p-4 sm:p-8 max-w-4xl mx-auto space-y-8 animate-fade-in">
      <div className="flex items-center justify-between border-b border-white/5 pb-4">
        <Link href="/" className="btn-dark !py-2 !px-4 text-xs font-semibold flex items-center gap-2">
          <ArrowLeft className="w-4 h-4" /> Voltar ao Início
        </Link>
        <span className="badge-ok text-xs">Em conformidade com a LGPD (Lei 13.709/2018)</span>
      </div>

      <div className="space-y-3">
        <div className="w-12 h-12 rounded-2xl bg-brand/15 text-brand flex items-center justify-center">
          <Shield className="w-6 h-6" />
        </div>
        <h1 className="text-3xl sm:text-4xl font-extrabold text-white tracking-tight">
          Política de Privacidade & Proteção de Dados
        </h1>
        <p className="text-sm text-ink-300">
          Última atualização: Outubro de 2026 · Plataforma Unika GymSync
        </p>
      </div>

      <div className="space-y-6 text-sm text-ink-200 leading-relaxed">
        <section className="card p-6 space-y-3">
          <h2 className="text-lg font-bold text-white flex items-center gap-2">
            <Lock className="w-5 h-5 text-brand" /> 1. Compromisso com sua Privacidade
          </h2>
          <p>
            A plataforma <strong>GymSync</strong> preza pela segurança, confidencialidade e integridade dos dados de suas alunas, professores e colaboradores. Tratamos seus dados pessoais em estrita observância à Lei Geral de Proteção de Dados Pessoais (LGPD - Lei nº 13.709/2018).
          </p>
        </section>

        <section className="card p-6 space-y-3">
          <h2 className="text-lg font-bold text-white flex items-center gap-2">
            <Database className="w-5 h-5 text-brand" /> 2. Dados Coletados e Finalidade
          </h2>
          <ul className="space-y-2 list-disc list-inside text-ink-300">
            <li><strong>Identificação e Cadastro:</strong> Nome completo, CPF, e-mail e telefone para identificação na entrada (check-in) e comunicação.</li>
            <li><strong>Saúde e Medidas Físicas:</strong> Peso, altura, circunferências corporais e histórico de treinos para prescrição individualizada de exercícios físicos por professores habilitados (Art. 7º e 11 da LGPD).</li>
            <li><strong>Financeiro e Pagamentos:</strong> Histórico de mensalidades e identificadores de transação PIX/Boleto. <em>Não armazenamos números de cartão de crédito em nossos servidores</em> — todas as operações utilizam tokenização segura do gateway de pagamento.</li>
          </ul>
        </section>

        <section className="card p-6 space-y-3">
          <h2 className="text-lg font-bold text-white flex items-center gap-2">
            <UserX className="w-5 h-5 text-brand" /> 3. Seus Direitos como Titular (Art. 18 LGPD)
          </h2>
          <p>Você tem o direito de, a qualquer momento e mediante requisição simples:</p>
          <div className="grid sm:grid-cols-2 gap-3 pt-2">
            <div className="p-3.5 rounded-2xl bg-ink-750 space-y-1">
              <strong className="text-white block">Acesso e Portabilidade</strong>
              <span className="text-xs text-ink-300">Consultar todos os seus dados e fichas de treino diretamente no aplicativo.</span>
            </div>
            <div className="p-3.5 rounded-2xl bg-ink-750 space-y-1">
              <strong className="text-white block">Correção e Atualização</strong>
              <span className="text-xs text-ink-300">Editar peso, medidas e dados cadastrais no seu perfil.</span>
            </div>
            <div className="p-3.5 rounded-2xl bg-ink-750 space-y-1">
              <strong className="text-white block">Eliminação de Dados</strong>
              <span className="text-xs text-ink-300">Solicitar exclusão da conta e anonimização dos dados pelo app (botão "Excluir Conta").</span>
            </div>
            <div className="p-3.5 rounded-2xl bg-ink-750 space-y-1">
              <strong className="text-white block">Revogação de Consentimento</strong>
              <span className="text-xs text-ink-300">Optar por não receber comunicações promocionais e avisos opcionais.</span>
            </div>
          </div>
        </section>

        <section className="card p-6 space-y-3">
          <h2 className="text-lg font-bold text-white flex items-center gap-2">
            <FileText className="w-5 h-5 text-brand" /> 4. Contato do Encarregado de Dados (DPO)
          </h2>
          <p>
            Para exercer seus direitos ou esclarecer qualquer dúvida sobre o tratamento de seus dados, entre em contato diretamente com nossa equipe de privacidade pelo e-mail: <strong className="text-brand">privacidade@unika.app</strong>.
          </p>
        </section>
      </div>

      <div className="text-center pt-4 text-xs text-ink-400">
        © 2026 Unika Fitness · GymSync Platform · Todos os direitos reservados.
      </div>
    </div>
  );
}
